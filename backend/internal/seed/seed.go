package seed

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"label3130/backend/internal/models"
)

func Run(db *gorm.DB, log *slog.Logger) error {
	classes := []string{"一班", "二班", "三班", "四班"}
	for _, name := range classes {
		if err := db.FirstOrCreate(&models.ClassRoom{}, models.ClassRoom{Name: name}).Error; err != nil {
			return fmt.Errorf("seed class %s: %w", name, err)
		}
	}

	if err := seedTeacher(db); err != nil {
		return err
	}
	if err := seedStudents(db); err != nil {
		return err
	}
	if err := seedQuestions(db); err != nil {
		return err
	}
	if err := seedPaperBlueprints(db); err != nil {
		return err
	}

	log.Info("seed completed")
	return nil
}

func seedTeacher(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.User{}).Where("role = ?", models.RoleTeacher).Count(&count).Error; err != nil {
		return fmt.Errorf("count teachers: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash teacher password: %w", err)
	}
	teacher := models.User{
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         models.RoleTeacher,
	}
	if err := db.Create(&teacher).Error; err != nil {
		return fmt.Errorf("create teacher: %w", err)
	}
	return nil
}

func seedStudents(db *gorm.DB) error {
	var classRoom models.ClassRoom
	if err := db.Where("name = ?", "一班").First(&classRoom).Error; err != nil {
		return fmt.Errorf("load class for student seed: %w", err)
	}

	students := []string{"stu001", "stu002"}
	for _, username := range students {
		var existing models.User
		err := db.Where("username = ?", username).First(&existing).Error
		if err == nil {
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check student %s: %w", username, err)
		}

		hash, hashErr := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		if hashErr != nil {
			return fmt.Errorf("hash student password: %w", hashErr)
		}
		user := models.User{
			Username:     username,
			PasswordHash: string(hash),
			Role:         models.RoleStudent,
			ClassID:      &classRoom.ID,
		}
		if createErr := db.Create(&user).Error; createErr != nil {
			return fmt.Errorf("create student %s: %w", username, createErr)
		}
	}
	return nil
}

func seedQuestions(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Question{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count questions: %w", err)
	}
	if count > 0 {
		return nil
	}

	templates := []models.Question{
		{
			Title:         "TCP 三次握手中用于建立连接的第二步是？",
			Description:   "网络基础",
			CreatedBy:     1,
			Difficulty:    models.DifficultyEasy,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "计算机网络,TCP/IP",
			Options: []models.QuestionOption{
				{Content: "客户端发送 SYN", IsCorrect: false},
				{Content: "服务端返回 SYN+ACK", IsCorrect: true},
				{Content: "客户端发送 FIN", IsCorrect: false},
				{Content: "服务端直接发送 ACK", IsCorrect: false},
			},
		},
		{
			Title:         "在 SQL 中用于去重查询结果的关键字是？",
			Description:   "数据库基础",
			CreatedBy:     1,
			Difficulty:    models.DifficultyEasy,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "数据库,SQL",
			Options: []models.QuestionOption{
				{Content: "ORDER BY", IsCorrect: false},
				{Content: "UNIQUE", IsCorrect: false},
				{Content: "DISTINCT", IsCorrect: true},
				{Content: "GROUP", IsCorrect: false},
			},
		},
		{
			Title:         "HTTP 状态码 404 表示？",
			Description:   "Web 基础",
			CreatedBy:     1,
			Difficulty:    models.DifficultyEasy,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "Web,HTTP",
			Options: []models.QuestionOption{
				{Content: "服务器内部错误", IsCorrect: false},
				{Content: "资源未找到", IsCorrect: true},
				{Content: "请求成功", IsCorrect: false},
				{Content: "未授权", IsCorrect: false},
			},
		},
		{
			Title:         "Git 用于查看提交历史的命令是？",
			Description:   "开发工具",
			CreatedBy:     1,
			Difficulty:    models.DifficultyEasy,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "Git,版本控制",
			Options: []models.QuestionOption{
				{Content: "git push", IsCorrect: false},
				{Content: "git log", IsCorrect: true},
				{Content: "git reset", IsCorrect: false},
				{Content: "git clean", IsCorrect: false},
			},
		},
		{
			Title:         "以下哪个不是 HTTP 请求方法？",
			Description:   "Web 基础",
			CreatedBy:     1,
			Difficulty:    models.DifficultyEasy,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "Web,HTTP",
			Options: []models.QuestionOption{
				{Content: "GET", IsCorrect: false},
				{Content: "POST", IsCorrect: false},
				{Content: "PUSH", IsCorrect: true},
				{Content: "DELETE", IsCorrect: false},
			},
		},
		{
			Title:         "数据库索引的主要作用是？",
			Description:   "数据库进阶",
			CreatedBy:     1,
			Difficulty:    models.DifficultyMedium,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "数据库,索引",
			Options: []models.QuestionOption{
				{Content: "节省存储空间", IsCorrect: false},
				{Content: "提高查询速度", IsCorrect: true},
				{Content: "保证数据完整性", IsCorrect: false},
				{Content: "实现数据加密", IsCorrect: false},
			},
		},
		{
			Title:         "TCP 和 UDP 的主要区别是？",
			Description:   "网络基础",
			CreatedBy:     1,
			Difficulty:    models.DifficultyMedium,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "计算机网络,TCP/IP,UDP",
			Options: []models.QuestionOption{
				{Content: "TCP 是面向连接的，UDP 是无连接的", IsCorrect: true},
				{Content: "TCP 比 UDP 更快", IsCorrect: false},
				{Content: "UDP 比 TCP 更可靠", IsCorrect: false},
				{Content: "TCP 和 UDP 没有区别", IsCorrect: false},
			},
		},
		{
			Title:         "以下哪些是 HTTP 状态码 2xx 的含义？",
			Description:   "Web 进阶",
			CreatedBy:     1,
			Difficulty:    models.DifficultyMedium,
			QuestionType:  models.QuestionTypeMultipleChoice,
			KnowledgeTags: "Web,HTTP",
			Options: []models.QuestionOption{
				{Content: "请求成功", IsCorrect: true},
				{Content: "请求被重定向", IsCorrect: false},
				{Content: "客户端错误", IsCorrect: false},
				{Content: "服务器错误", IsCorrect: false},
			},
		},
		{
			Title:         "解释一下什么是 RESTful API？",
			Description:   "Web 架构",
			CreatedBy:     1,
			Difficulty:    models.DifficultyMedium,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "Web,API,架构",
			Options: []models.QuestionOption{
				{Content: "一种数据库设计规范", IsCorrect: false},
				{Content: "一种基于 HTTP 的 API 设计风格", IsCorrect: true},
				{Content: "一种编程语言", IsCorrect: false},
				{Content: "一种网络协议", IsCorrect: false},
			},
		},
		{
			Title:         "B+ 树和 B 树的区别是什么？",
			Description:   "数据结构与数据库",
			CreatedBy:     1,
			Difficulty:    models.DifficultyHard,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "数据结构,数据库,索引",
			Options: []models.QuestionOption{
				{Content: "B+ 树只有叶子节点存数据，B 树所有节点都存", IsCorrect: true},
				{Content: "B 树只有叶子节点存数据，B+ 树所有节点都存", IsCorrect: false},
				{Content: "B+ 树是二叉树，B 树是多叉树", IsCorrect: false},
				{Content: "两者没有区别", IsCorrect: false},
			},
		},
		{
			Title:         "什么是数据库事务的 ACID 特性？",
			Description:   "数据库进阶",
			CreatedBy:     1,
			Difficulty:    models.DifficultyHard,
			QuestionType:  models.QuestionTypeMultipleChoice,
			KnowledgeTags: "数据库,事务",
			Options: []models.QuestionOption{
				{Content: "原子性(Atomicity)", IsCorrect: true},
				{Content: "一致性(Consistency)", IsCorrect: true},
				{Content: "隔离性(Isolation)", IsCorrect: true},
				{Content: "持久性(Durability)", IsCorrect: true},
			},
		},
		{
			Title:         "请解释 HTTPS 的工作原理",
			Description:   "网络安全",
			CreatedBy:     1,
			Difficulty:    models.DifficultyHard,
			QuestionType:  models.QuestionTypeSingleChoice,
			KnowledgeTags: "计算机网络,安全,HTTPS",
			Options: []models.QuestionOption{
				{Content: "通过 SSL/TLS 加密 HTTP 通信", IsCorrect: true},
				{Content: "只是 HTTP 的更快版本", IsCorrect: false},
				{Content: "使用 UDP 协议传输", IsCorrect: false},
				{Content: "不需要证书验证", IsCorrect: false},
			},
		},
		{
			Title:         "判断题：HTTP 是无状态协议。",
			Description:   "Web 基础",
			CreatedBy:     1,
			Difficulty:    models.DifficultyEasy,
			QuestionType:  models.QuestionTypeTrueFalse,
			KnowledgeTags: "Web,HTTP",
			Options: []models.QuestionOption{
				{Content: "正确", IsCorrect: true},
				{Content: "错误", IsCorrect: false},
			},
		},
		{
			Title:         "判断题：MySQL 中的 MyISAM 引擎支持事务。",
			Description:   "数据库进阶",
			CreatedBy:     1,
			Difficulty:    models.DifficultyMedium,
			QuestionType:  models.QuestionTypeTrueFalse,
			KnowledgeTags: "数据库,MySQL",
			Options: []models.QuestionOption{
				{Content: "正确", IsCorrect: false},
				{Content: "错误", IsCorrect: true},
			},
		},
		{
			Title:         "以下哪些属于前端框架？",
			Description:   "前端开发",
			CreatedBy:     1,
			Difficulty:    models.DifficultyEasy,
			QuestionType:  models.QuestionTypeMultipleChoice,
			KnowledgeTags: "前端,框架",
			Options: []models.QuestionOption{
				{Content: "React", IsCorrect: true},
				{Content: "Vue", IsCorrect: true},
				{Content: "Django", IsCorrect: false},
				{Content: "Angular", IsCorrect: true},
			},
		},
	}

	for _, item := range templates {
		if err := db.Create(&item).Error; err != nil {
			return fmt.Errorf("seed question: %w", err)
		}
	}
	return nil
}

func seedPaperBlueprints(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.PaperBlueprint{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count blueprints: %w", err)
	}
	if count > 0 {
		return nil
	}

	rule := models.PaperRule{
		TotalQuestions:   10,
		PerQuestionScore: 10,
		Difficulty: []models.DifficultyRule{
			{Level: models.DifficultyEasy, Count: 3, Ratio: 0.3},
			{Level: models.DifficultyMedium, Count: 5, Ratio: 0.5},
			{Level: models.DifficultyHard, Count: 2, Ratio: 0.2},
		},
		QuestionTypes: []models.QuestionTypeRule{
			{Type: models.QuestionTypeSingleChoice, Count: 7, Ratio: 0.7},
			{Type: models.QuestionTypeMultipleChoice, Count: 2, Ratio: 0.2},
			{Type: models.QuestionTypeTrueFalse, Count: 1, Ratio: 0.1},
		},
	}

	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("marshal rule: %w", err)
	}

	blueprint := models.PaperBlueprint{
		Name:            "综合测试卷模板",
		Description:     "包含易、中、难三种难度的综合测试",
		TotalScore:      100,
		RuleJSON:        string(ruleJSON),
		AvoidRepeatDays: 7,
		CreatedBy:       1,
	}

	if err := db.Create(&blueprint).Error; err != nil {
		return fmt.Errorf("seed blueprint: %w", err)
	}

	return nil
}
