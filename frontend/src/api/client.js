const API_BASE = import.meta.env.VITE_API_BASE || '/api';

async function parseResponse(response) {
  const contentType = response.headers.get('content-type') || '';
  const isJSON = contentType.includes('application/json');
  const payload = isJSON ? await response.json() : null;

  if (!response.ok) {
    const message = payload?.message || `Request failed: ${response.status}`;
    throw new Error(message);
  }
  return payload;
}

export async function apiRequest(path, { method = 'GET', token, body, isForm = false } = {}) {
  const headers = {};
  if (!isForm) {
    headers['Content-Type'] = 'application/json';
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: isForm ? body : body ? JSON.stringify(body) : undefined,
  });

  return parseResponse(response);
}

export async function getSubjectiveQuestions(token) {
  return apiRequest('/teacher/subjective-questions', { token });
}

export async function getSubjectiveQuestion(id, token) {
  return apiRequest(`/teacher/subjective-questions/${id}`, { token });
}

export async function createSubjectiveQuestion(data, token) {
  return apiRequest('/teacher/subjective-questions', { method: 'POST', token, body: data });
}

export async function updateSubjectiveQuestion(id, data, token) {
  return apiRequest(`/teacher/subjective-questions/${id}`, { method: 'PUT', token, body: data });
}

export async function deleteSubjectiveQuestion(id, token) {
  return apiRequest(`/teacher/subjective-questions/${id}`, { method: 'DELETE', token });
}

export async function getSubjectiveSubmissions(params, token) {
  const query = new URLSearchParams();
  if (params.classId) query.set('classId', params.classId);
  if (params.questionId) query.set('questionId', params.questionId);
  if (params.status) query.set('status', params.status);
  if (params.page) query.set('page', params.page);
  if (params.pageSize) query.set('pageSize', params.pageSize);
  return apiRequest(`/teacher/subjective-submissions?${query.toString()}`, { token });
}

export async function getSubjectiveSubmission(id, token) {
  return apiRequest(`/teacher/subjective-submissions/${id}`, { token });
}

export async function gradeSubjectiveSubmission(id, data, token) {
  return apiRequest(`/teacher/subjective-submissions/${id}/grade`, { method: 'POST', token, body: data });
}

export async function getSubjectivePendingCount(token) {
  return apiRequest('/teacher/subjective-pending-count', { token });
}

export async function getStudentSubjectiveQuestions(token) {
  return apiRequest('/student/subjective-questions', { token });
}

export async function getStudentSubjectiveQuestion(id, token) {
  return apiRequest(`/student/subjective-questions/${id}`, { token });
}

export async function submitSubjectiveAnswer(data, token) {
  return apiRequest('/student/subjective-submit', { method: 'POST', token, body: data });
}

export async function getStudentSubjectiveSubmissions(token) {
  return apiRequest('/student/subjective-submissions', { token });
}

export async function getStudentSubjectiveSubmission(id, token) {
  return apiRequest(`/student/subjective-submissions/${id}`, { token });
}

export async function getExams(params, token) {
  const query = new URLSearchParams();
  if (params?.status) query.set('status', params.status);
  if (params?.classId) query.set('classId', params.classId);
  if (params?.page) query.set('page', params.page);
  if (params?.pageSize) query.set('pageSize', params.pageSize);
  return apiRequest(`/teacher/exams?${query.toString()}`, { token });
}

export async function getExam(id, token) {
  return apiRequest(`/teacher/exams/${id}`, { token });
}

export async function createExam(data, token) {
  return apiRequest('/teacher/exams', { method: 'POST', token, body: data });
}

export async function updateExam(id, data, token) {
  return apiRequest(`/teacher/exams/${id}`, { method: 'PUT', token, body: data });
}

export async function deleteExam(id, token) {
  return apiRequest(`/teacher/exams/${id}`, { method: 'DELETE', token });
}

export async function getExamParticipants(id, token) {
  return apiRequest(`/teacher/exams/${id}/participants`, { token });
}

export async function getStudentExams(token) {
  return apiRequest('/student/exams', { token });
}

export async function getStudentExamDetail(id, token) {
  return apiRequest(`/student/exams/${id}`, { token });
}

export async function enterExam(id, token) {
  return apiRequest(`/student/exams/${id}/enter`, { method: 'POST', token });
}

export async function submitExam(id, answers, token) {
  return apiRequest(`/student/exams/${id}/submit`, { method: 'POST', token, body: { answers } });
}

export async function getExamResult(id, token) {
  return apiRequest(`/student/exams/${id}/result`, { token });
}
