package service

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"label3130/backend/internal/dto"
	"label3130/backend/internal/models"
)

const (
	wsMsgTypeWelcome      = "welcome"
	wsMsgTypeRoomInfo     = "room_info"
	wsMsgTypeOpponentJoin = "opponent_join"
	wsMsgTypeGameStart    = "game_start"
	wsMsgTypeRoundStart   = "round_start"
	wsMsgTypeRoundResult  = "round_result"
	wsMsgTypeGameOver     = "game_over"
	wsMsgTypePlayerLeave  = "player_leave"
	wsMsgTypeReconnect    = "reconnect"
	wsMsgTypeAnswer       = "answer"
	wsMsgTypeError        = "error"
)

var (
	ErrPkRoomNotFound   = errors.New("pk room not found")
	ErrPkRoomFull       = errors.New("pk room is full")
	ErrPkRoomNotWaiting = errors.New("pk room is not waiting")
	ErrPkAlreadyInRoom  = errors.New("you are already in this room")
	ErrPkNotInRoom      = errors.New("you are not in this room")
	ErrPkGameEnded      = errors.New("pk game has ended")
	ErrPkInvalidAnswer  = errors.New("invalid answer")
)

type pkClient struct {
	UserID   uint
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Room     *pkRoomState
	IsPlayerA bool
}

type pkRoomState struct {
	RoomID     uint
	RoomCode   string
	QuestionCount int
	TimePerQuestion int
	PlayerA    *pkClient
	PlayerB    *pkClient
	Questions  []models.Question
	ScoreA     int
	ScoreB     int
	RoundIndex int
	RoundStartAt time.Time
	PlayerAAnswer *answerRecord
	PlayerBAnswer *answerRecord
	Status     string
	mu         sync.Mutex
	timer      *time.Timer
	Results    []models.PkRoundResult
	Finished   bool
}

type answerRecord struct {
	OptionID  uint
	IsCorrect bool
	TimeMs    int
	Answered  bool
}

type PkService struct {
	db     *gorm.DB
	log    *slog.Logger
	rooms  map[string]*pkRoomState
	roomsMu sync.RWMutex
}

func NewPkService(db *gorm.DB, log *slog.Logger) *PkService {
	return &PkService{
		db:    db,
		log:   log,
		rooms: make(map[string]*pkRoomState),
	}
}

func generateRoomCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		code[i] = chars[n.Int64()]
	}
	return string(code), nil
}

func (s *PkService) CreateRoom(userID uint, username string, req dto.CreatePkRoomRequest) (*dto.PkRoomResponse, error) {
	questionCount := req.QuestionCount
	if questionCount == 0 {
		questionCount = 10
	}
	timePerQuestion := req.TimePerQuestion
	if timePerQuestion == 0 {
		timePerQuestion = 15
	}

	var questions []models.Question
	if err := s.db.Preload("Options").
		Order("RAND()").
		Limit(questionCount).
		Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("load questions: %w", err)
	}
	if len(questions) < questionCount {
		return nil, ErrNoQuestions
	}

	questionsJSON, err := json.Marshal(questions)
	if err != nil {
		return nil, fmt.Errorf("marshal questions: %w", err)
	}

	var roomCode string
	for i := 0; i < 10; i++ {
		code, err := generateRoomCode()
		if err != nil {
			return nil, err
		}
		var count int64
		s.db.Model(&models.PkRoom{}).Where("room_code = ?", code).Count(&count)
		if count == 0 {
			roomCode = code
			break
		}
	}
	if roomCode == "" {
		return nil, fmt.Errorf("failed to generate unique room code")
	}

	room := &models.PkRoom{
		RoomCode:      roomCode,
		Status:        models.PkRoomStatusWaiting,
		QuestionCount: questionCount,
		TimePerQuestion: timePerQuestion,
		PlayerAID:     &userID,
		Questions:     string(questionsJSON),
	}

	if err := s.db.Create(room).Error; err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}

	if err := s.db.Preload("PlayerA").First(room, room.ID).Error; err != nil {
		return nil, fmt.Errorf("load room: %w", err)
	}

	state := &pkRoomState{
		RoomID:          room.ID,
		RoomCode:        roomCode,
		QuestionCount:   questionCount,
		TimePerQuestion: timePerQuestion,
		Questions:       questions,
		ScoreA:          0,
		ScoreB:          0,
		RoundIndex:      -1,
		Status:          models.PkRoomStatusWaiting,
	}

	playerA := &pkClient{
		UserID:    userID,
		Username:  username,
		Send:      make(chan []byte, 256),
		Room:      state,
		IsPlayerA: true,
	}
	state.PlayerA = playerA

	s.roomsMu.Lock()
	s.rooms[roomCode] = state
	s.roomsMu.Unlock()

	return s.toRoomResponse(room), nil
}

func (s *PkService) JoinRoom(userID uint, username string, req dto.JoinPkRoomRequest) (*dto.PkRoomResponse, error) {
	s.roomsMu.RLock()
	state, exists := s.rooms[req.RoomCode]
	s.roomsMu.RUnlock()

	if !exists {
		var room models.PkRoom
		if err := s.db.Preload("PlayerA").Preload("PlayerB").
			Where("room_code = ?", req.RoomCode).
			First(&room).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPkRoomNotFound
			}
			return nil, fmt.Errorf("load room: %w", err)
		}

		if room.Status != models.PkRoomStatusWaiting {
			return nil, ErrPkRoomNotWaiting
		}
		if room.PlayerBID != nil {
			return nil, ErrPkRoomFull
		}
		if room.PlayerAID != nil && *room.PlayerAID == userID {
			return nil, ErrPkAlreadyInRoom
		}

		var questions []models.Question
		if err := json.Unmarshal([]byte(room.Questions), &questions); err != nil {
			return nil, fmt.Errorf("unmarshal questions: %w", err)
		}

		state = &pkRoomState{
			RoomID:          room.ID,
			RoomCode:        room.RoomCode,
			QuestionCount:   room.QuestionCount,
			TimePerQuestion: room.TimePerQuestion,
			Questions:       questions,
			ScoreA:          room.ScoreA,
			ScoreB:          room.ScoreB,
			RoundIndex:      -1,
			Status:          room.Status,
		}

		playerA := &pkClient{
			UserID:    *room.PlayerAID,
			Username:  room.PlayerA.Username,
			Send:      make(chan []byte, 256),
			Room:      state,
			IsPlayerA: true,
		}
		state.PlayerA = playerA

		s.roomsMu.Lock()
		s.rooms[req.RoomCode] = state
		s.roomsMu.Unlock()
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.Status != models.PkRoomStatusWaiting {
		return nil, ErrPkRoomNotWaiting
	}
	if state.PlayerB != nil {
		return nil, ErrPkRoomFull
	}
	if state.PlayerA != nil && state.PlayerA.UserID == userID {
		return nil, ErrPkAlreadyInRoom
	}

	playerB := &pkClient{
		UserID:    userID,
		Username:  username,
		Send:      make(chan []byte, 256),
		Room:      state,
		IsPlayerA: false,
	}
	state.PlayerB = playerB

	if err := s.db.Model(&models.PkRoom{}).
		Where("id = ?", state.RoomID).
		Update("player_b_id", userID).Error; err != nil {
		return nil, fmt.Errorf("update room: %w", err)
	}

	go s.broadcastRoomInfo(state)

	var room models.PkRoom
	if err := s.db.Preload("PlayerA").Preload("PlayerB").
		First(&room, state.RoomID).Error; err == nil {
		return s.toRoomResponse(&room), nil
	}

	return &dto.PkRoomResponse{
		ID:              state.RoomID,
		RoomCode:        state.RoomCode,
		Status:          state.Status,
		QuestionCount:   state.QuestionCount,
		TimePerQuestion: state.TimePerQuestion,
		PlayerAID:       &state.PlayerA.UserID,
		PlayerAName:     state.PlayerA.Username,
		PlayerBID:       &userID,
		PlayerBName:     username,
		ScoreA:          state.ScoreA,
		ScoreB:          state.ScoreB,
	}, nil
}

func (s *PkService) GetRoomInfo(roomCode string) (*dto.PkRoomResponse, error) {
	var room models.PkRoom
	if err := s.db.Preload("PlayerA").Preload("PlayerB").
		Where("room_code = ?", roomCode).
		First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPkRoomNotFound
		}
		return nil, fmt.Errorf("load room: %w", err)
	}
	return s.toRoomResponse(&room), nil
}

func (s *PkService) HandleWebSocket(conn *websocket.Conn, userID uint, username string, roomCode string) {
	s.roomsMu.RLock()
	state, exists := s.rooms[roomCode]
	s.roomsMu.RUnlock()

	if !exists {
		conn.WriteJSON(dto.PkWsMessage{Type: wsMsgTypeError, Data: "room not found"})
		conn.Close()
		return
	}

	var client *pkClient
	var isPlayerA bool

	state.mu.Lock()
	if state.PlayerA != nil && state.PlayerA.UserID == userID {
		if state.PlayerA.Conn != nil {
			state.PlayerA.Conn.Close()
		}
		client = state.PlayerA
		client.Conn = conn
		isPlayerA = true
	} else if state.PlayerB != nil && state.PlayerB.UserID == userID {
		if state.PlayerB.Conn != nil {
			state.PlayerB.Conn.Close()
		}
		client = state.PlayerB
		client.Conn = conn
		isPlayerA = false
	} else {
		state.mu.Unlock()
		conn.WriteJSON(dto.PkWsMessage{Type: wsMsgTypeError, Data: "you are not in this room"})
		conn.Close()
		return
	}
	state.mu.Unlock()

	defer func() {
		state.mu.Lock()
		if client.Conn == conn {
			client.Conn = nil
		}
		state.mu.Unlock()

		go s.handlePlayerDisconnect(state, userID)
	}()

	welcomeMsg := dto.PkWsMessage{
		Type: wsMsgTypeWelcome,
		Data: map[string]interface{}{
			"roomId":     state.RoomID,
			"roomCode":   state.RoomCode,
			"isPlayerA":  isPlayerA,
			"status":     state.Status,
			"roundIndex": state.RoundIndex,
			"scoreA":     state.ScoreA,
			"scoreB":     state.ScoreB,
		},
	}
	conn.WriteJSON(welcomeMsg)

	if state.Status == models.PkRoomStatusOngoing && state.RoundIndex >= 0 {
		s.sendCurrentRound(state, client)
	}

	go s.writePump(client)
	s.readPump(client, isPlayerA)
}

func (s *PkService) readPump(client *pkClient, isPlayerA bool) {
	defer client.Conn.Close()

	for {
		var msg dto.PkWsMessage
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				s.log.Warn("pk ws read error", "error", err.Error(), "userID", client.UserID)
			}
			break
		}

		switch msg.Type {
		case wsMsgTypeAnswer:
			dataMap, ok := msg.Data.(map[string]interface{})
			if !ok {
				continue
			}
			questionID := uint(dataMap["questionId"].(float64))
			optionID := uint(dataMap["optionId"].(float64))
			s.handleAnswer(client.Room, client, isPlayerA, questionID, optionID)

		case wsMsgTypeGameStart:
			if isPlayerA && client.Room.Status == models.PkRoomStatusWaiting && client.Room.PlayerB != nil {
				go s.startGame(client.Room)
			}
		}
	}
}

func (s *PkService) writePump(client *pkClient) {
	defer client.Conn.Close()

	for msg := range client.Send {
		client.Conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func (s *PkService) handleAnswer(room *pkRoomState, client *pkClient, isPlayerA bool, questionID uint, optionID uint) {
	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Status != models.PkRoomStatusOngoing {
		return
	}
	if room.RoundIndex < 0 || room.RoundIndex >= len(room.Questions) {
		return
	}
	currentQ := room.Questions[room.RoundIndex]
	if currentQ.ID != questionID {
		return
	}

	var record *answerRecord
	if isPlayerA {
		if room.PlayerAAnswer != nil && room.PlayerAAnswer.Answered {
			return
		}
		room.PlayerAAnswer = &answerRecord{Answered: true}
		record = room.PlayerAAnswer
	} else {
		if room.PlayerBAnswer != nil && room.PlayerBAnswer.Answered {
			return
		}
		room.PlayerBAnswer = &answerRecord{Answered: true}
		record = room.PlayerBAnswer
	}

	elapsed := time.Since(room.RoundStartAt)
	record.TimeMs = int(elapsed.Milliseconds())
	record.OptionID = optionID

	for _, opt := range currentQ.Options {
		if opt.ID == optionID {
			record.IsCorrect = opt.IsCorrect
			break
		}
	}

	if record.IsCorrect {
		bonus := 100
		if record.TimeMs < 3000 {
			bonus = 100
		} else if record.TimeMs < 5000 {
			bonus = 80
		} else if record.TimeMs < 8000 {
			bonus = 60
		} else {
			bonus = 40
		}
		if isPlayerA {
			room.ScoreA += bonus
		} else {
			room.ScoreB += bonus
		}
	}

	bothAnswered := (room.PlayerAAnswer != nil && room.PlayerAAnswer.Answered) &&
		(room.PlayerBAnswer != nil && room.PlayerBAnswer.Answered)

	if bothAnswered {
		if room.timer != nil {
			room.timer.Stop()
			room.timer = nil
		}
		go s.finishRound(room)
	}
}

func (s *PkService) startGame(room *pkRoomState) {
	room.mu.Lock()
	if room.Status != models.PkRoomStatusWaiting || room.PlayerA == nil || room.PlayerB == nil {
		room.mu.Unlock()
		return
	}
	room.Status = models.PkRoomStatusOngoing
	now := time.Now()

	s.db.Model(&models.PkRoom{}).
		Where("id = ?", room.RoomID).
		Updates(map[string]interface{}{
			"status":     models.PkRoomStatusOngoing,
			"started_at": now,
		})
	room.mu.Unlock()

	startMsg := dto.PkWsMessage{
		Type: wsMsgTypeGameStart,
		Data: map[string]interface{}{
			"questionCount":   room.QuestionCount,
			"timePerQuestion": room.TimePerQuestion,
			"playerAName":     room.PlayerA.Username,
			"playerBName":     room.PlayerB.Username,
		},
	}
	s.broadcastMessage(room, startMsg)

	time.Sleep(2 * time.Second)
	s.nextRound(room)
}

func (s *PkService) nextRound(room *pkRoomState) {
	room.mu.Lock()
	if room.Finished || room.Status != models.PkRoomStatusOngoing {
		room.mu.Unlock()
		return
	}

	room.RoundIndex++
	if room.RoundIndex >= len(room.Questions) {
		room.mu.Unlock()
		s.endGame(room)
		return
	}

	room.RoundStartAt = time.Now()
	room.PlayerAAnswer = nil
	room.PlayerBAnswer = nil

	q := room.Questions[room.RoundIndex]
	options := make([]dto.PkOptionBrief, len(q.Options))
	for i, opt := range q.Options {
		options[i] = dto.PkOptionBrief{
			ID:      opt.ID,
			Content: opt.Content,
		}
	}

	roundData := dto.PkRoundData{
		RoundIndex:   room.RoundIndex,
		QuestionID:   q.ID,
		Title:        q.Title,
		Options:      options,
		TimeLimitSec: room.TimePerQuestion,
		StartAt:      room.RoundStartAt.UnixMilli(),
	}

	roundMsg := dto.PkWsMessage{
		Type: wsMsgTypeRoundStart,
		Data: roundData,
	}
	s.broadcastMessage(room, roundMsg)

	room.timer = time.AfterFunc(time.Duration(room.TimePerQuestion)*time.Second, func() {
		room.mu.Lock()
		if room.timer != nil {
			room.timer = nil
			room.mu.Unlock()
			s.finishRound(room)
		} else {
			room.mu.Unlock()
		}
	})
	room.mu.Unlock()
}

func (s *PkService) finishRound(room *pkRoomState) {
	room.mu.Lock()
	if room.Finished || room.RoundIndex < 0 || room.RoundIndex >= len(room.Questions) {
		room.mu.Unlock()
		return
	}

	if room.timer != nil {
		room.timer.Stop()
		room.timer = nil
	}

	q := room.Questions[room.RoundIndex]

	var correctOptionID uint
	for _, opt := range q.Options {
		if opt.IsCorrect {
			correctOptionID = opt.ID
			break
		}
	}

	playerACorrect := false
	playerATimeMs := 0
	if room.PlayerAAnswer != nil && room.PlayerAAnswer.Answered {
		playerACorrect = room.PlayerAAnswer.IsCorrect
		playerATimeMs = room.PlayerAAnswer.TimeMs
	}

	playerBCorrect := false
	playerBTimeMs := 0
	if room.PlayerBAnswer != nil && room.PlayerBAnswer.Answered {
		playerBCorrect = room.PlayerBAnswer.IsCorrect
		playerBTimeMs = room.PlayerBAnswer.TimeMs
	}

	roundResult := models.PkRoundResult{
		RoomID:     room.RoomID,
		RoundIndex: room.RoundIndex,
		QuestionID: q.ID,
	}
	if room.PlayerAAnswer != nil && room.PlayerAAnswer.Answered {
		roundResult.PlayerAOptionID = &room.PlayerAAnswer.OptionID
		roundResult.PlayerAIsCorrect = &room.PlayerAAnswer.IsCorrect
		roundResult.PlayerATimeMs = &room.PlayerAAnswer.TimeMs
	}
	if room.PlayerBAnswer != nil && room.PlayerBAnswer.Answered {
		roundResult.PlayerBOptionID = &room.PlayerBAnswer.OptionID
		roundResult.PlayerBIsCorrect = &room.PlayerBAnswer.IsCorrect
		roundResult.PlayerBTimeMs = &room.PlayerBAnswer.TimeMs
	}
	room.Results = append(room.Results, roundResult)

	resultData := dto.PkRoundResultData{
		RoundIndex:      room.RoundIndex,
		QuestionID:      q.ID,
		CorrectOptionID: correctOptionID,
		PlayerACorrect:  playerACorrect,
		PlayerBCorrect:  playerBCorrect,
		PlayerATimeMs:   playerATimeMs,
		PlayerBTimeMs:   playerBTimeMs,
		ScoreA:          room.ScoreA,
		ScoreB:          room.ScoreB,
	}

	resultMsg := dto.PkWsMessage{
		Type: wsMsgTypeRoundResult,
		Data: resultData,
	}
	s.broadcastMessage(room, resultMsg)
	room.mu.Unlock()

	time.Sleep(2500 * time.Millisecond)
	s.nextRound(room)
}

func (s *PkService) endGame(room *pkRoomState) {
	room.mu.Lock()
	if room.Finished {
		room.mu.Unlock()
		return
	}
	room.Finished = true
	room.Status = models.PkRoomStatusFinished

	var winnerID *uint
	var winnerName string
	isDraw := false

	if room.ScoreA > room.ScoreB {
		winnerID = &room.PlayerA.UserID
		winnerName = room.PlayerA.Username
	} else if room.ScoreB > room.ScoreA {
		winnerID = &room.PlayerB.UserID
		winnerName = room.PlayerB.Username
	} else {
		isDraw = true
	}

	now := time.Now()
	s.db.Model(&models.PkRoom{}).
		Where("id = ?", room.RoomID).
		Updates(map[string]interface{}{
			"status":      models.PkRoomStatusFinished,
			"score_a":     room.ScoreA,
			"score_b":     room.ScoreB,
			"winner_id":   winnerID,
			"finished_at": now,
		})

	if len(room.Results) > 0 {
		s.db.Create(&room.Results)
	}

	gameOverData := dto.PkGameOverData{
		IsDraw: isDraw,
		ScoreA: room.ScoreA,
		ScoreB: room.ScoreB,
	}
	if winnerID != nil {
		gameOverData.WinnerID = *winnerID
		gameOverData.WinnerName = winnerName
	}

	gameOverMsg := dto.PkWsMessage{
		Type: wsMsgTypeGameOver,
		Data: gameOverData,
	}
	s.broadcastMessage(room, gameOverMsg)
	room.mu.Unlock()

	go func() {
		time.Sleep(30 * time.Second)
		s.roomsMu.Lock()
		if r, ok := s.rooms[room.RoomCode]; ok && r == room {
			delete(s.rooms, room.RoomCode)
		}
		s.roomsMu.Unlock()
	}()
}

func (s *PkService) handlePlayerDisconnect(room *pkRoomState, userID uint) {
	time.Sleep(5 * time.Second)

	room.mu.Lock()
	defer room.mu.Unlock()

	var player *pkClient
	var isPlayerA bool
	if room.PlayerA != nil && room.PlayerA.UserID == userID {
		player = room.PlayerA
		isPlayerA = true
	} else if room.PlayerB != nil && room.PlayerB.UserID == userID {
		player = room.PlayerB
		isPlayerA = false
	} else {
		return
	}

	if player.Conn != nil {
		return
	}

	if room.Status == models.PkRoomStatusWaiting {
		return
	}

	if room.Status == models.PkRoomStatusOngoing && !room.Finished {
		room.Finished = true
		room.Status = models.PkRoomStatusFinished

		var winnerID uint
		var winnerName string
		if isPlayerA {
			room.ScoreB += 500
			winnerID = room.PlayerB.UserID
			winnerName = room.PlayerB.Username
		} else {
			room.ScoreA += 500
			winnerID = room.PlayerA.UserID
			winnerName = room.PlayerA.Username
		}

		now := time.Now()
		s.db.Model(&models.PkRoom{}).
			Where("id = ?", room.RoomID).
			Updates(map[string]interface{}{
				"status":      models.PkRoomStatusFinished,
				"score_a":     room.ScoreA,
				"score_b":     room.ScoreB,
				"winner_id":   winnerID,
				"finished_at": now,
			})

		leaveMsg := dto.PkWsMessage{
			Type: wsMsgTypePlayerLeave,
			Data: dto.PkPlayerLeaveData{
				PlayerID:   userID,
				PlayerName: player.Username,
				Reason:     "对手已离场，你获胜了！",
			},
		}

		if isPlayerA && room.PlayerB != nil && room.PlayerB.Conn != nil {
			room.PlayerB.Send <- mustMarshal(leaveMsg)
		} else if !isPlayerA && room.PlayerA != nil && room.PlayerA.Conn != nil {
			room.PlayerA.Send <- mustMarshal(leaveMsg)
		}

		gameOverData := dto.PkGameOverData{
			WinnerID:   winnerID,
			WinnerName: winnerName,
			IsDraw:     false,
			ScoreA:     room.ScoreA,
			ScoreB:     room.ScoreB,
		}
		gameOverMsg := dto.PkWsMessage{
			Type: wsMsgTypeGameOver,
			Data: gameOverData,
		}

		if isPlayerA && room.PlayerB != nil && room.PlayerB.Conn != nil {
			room.PlayerB.Send <- mustMarshal(gameOverMsg)
		} else if !isPlayerA && room.PlayerA != nil && room.PlayerA.Conn != nil {
			room.PlayerA.Send <- mustMarshal(gameOverMsg)
		}
	}
}

func (s *PkService) sendCurrentRound(room *pkRoomState, client *pkClient) {
	room.mu.Lock()
	defer room.mu.Unlock()

	if room.RoundIndex < 0 || room.RoundIndex >= len(room.Questions) {
		return
	}

	q := room.Questions[room.RoundIndex]
	options := make([]dto.PkOptionBrief, len(q.Options))
	for i, opt := range q.Options {
		options[i] = dto.PkOptionBrief{
			ID:      opt.ID,
			Content: opt.Content,
		}
	}

	elapsed := time.Since(room.RoundStartAt)
	remaining := time.Duration(room.TimePerQuestion)*time.Second - elapsed
	if remaining < 0 {
		remaining = 0
	}

	roundData := dto.PkRoundData{
		RoundIndex:   room.RoundIndex,
		QuestionID:   q.ID,
		Title:        q.Title,
		Options:      options,
		TimeLimitSec: int(remaining.Seconds()),
		StartAt:      room.RoundStartAt.UnixMilli(),
	}

	msg := dto.PkWsMessage{
		Type: wsMsgTypeRoundStart,
		Data: roundData,
	}
	client.Send <- mustMarshal(msg)
}

func (s *PkService) broadcastRoomInfo(room *pkRoomState) {
	room.mu.Lock()
	defer room.mu.Unlock()

	info := map[string]interface{}{
		"roomCode":   room.RoomCode,
		"status":     room.Status,
		"scoreA":     room.ScoreA,
		"scoreB":     room.ScoreB,
		"playerA":    nil,
		"playerB":    nil,
	}
	if room.PlayerA != nil {
		info["playerA"] = map[string]interface{}{
			"id":       room.PlayerA.UserID,
			"username": room.PlayerA.Username,
		}
	}
	if room.PlayerB != nil {
		info["playerB"] = map[string]interface{}{
			"id":       room.PlayerB.UserID,
			"username": room.PlayerB.Username,
		}
	}

	msg := dto.PkWsMessage{Type: wsMsgTypeRoomInfo, Data: info}
	s.broadcastMessage(room, msg)
}

func (s *PkService) broadcastMessage(room *pkRoomState, msg dto.PkWsMessage) {
	data := mustMarshal(msg)
	if room.PlayerA != nil && room.PlayerA.Conn != nil {
		select {
		case room.PlayerA.Send <- data:
		default:
		}
	}
	if room.PlayerB != nil && room.PlayerB.Conn != nil {
		select {
		case room.PlayerB.Send <- data:
		default:
		}
	}
}

func (s *PkService) toRoomResponse(room *models.PkRoom) *dto.PkRoomResponse {
	resp := &dto.PkRoomResponse{
		ID:              room.ID,
		RoomCode:        room.RoomCode,
		Status:          room.Status,
		QuestionCount:   room.QuestionCount,
		TimePerQuestion: room.TimePerQuestion,
		ScoreA:          room.ScoreA,
		ScoreB:          room.ScoreB,
	}
	if room.PlayerAID != nil {
		resp.PlayerAID = room.PlayerAID
		if room.PlayerA != nil {
			resp.PlayerAName = room.PlayerA.Username
		}
	}
	if room.PlayerBID != nil {
		resp.PlayerBID = room.PlayerBID
		if room.PlayerB != nil {
			resp.PlayerBName = room.PlayerB.Username
		}
	}
	if room.WinnerID != nil {
		resp.WinnerID = room.WinnerID
	}
	if room.StartedAt != nil {
		resp.StartedAt = room.StartedAt.Format(time.RFC3339)
	}
	if room.FinishedAt != nil {
		resp.FinishedAt = room.FinishedAt.Format(time.RFC3339)
	}
	return resp
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func (s *PkService) GetRoundResults(roomID uint) ([]models.PkRoundResult, error) {
	var results []models.PkRoundResult
	if err := s.db.Where("room_id = ?", roomID).Order("round_index asc").Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (s *PkService) GetRoomByID(roomID uint) (*models.PkRoom, error) {
	var room models.PkRoom
	if err := s.db.Preload("PlayerA").Preload("PlayerB").Preload("Winner").
		First(&room, roomID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPkRoomNotFound
		}
		return nil, err
	}
	return &room, nil
}
