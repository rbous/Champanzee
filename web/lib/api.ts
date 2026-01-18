const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/v1';

// ============================================
// Types
// ============================================

export interface LoginResponse {
    token: string;
}

export interface Survey {
    id: string;
    title: string;
    intent: string;
    settings: SurveySettings;
    questions: Question[];
    createdAt: string;
}

export interface SurveySettings {
    maxFollowUps: number;
    allowSkipAfter: number;
}

export interface Question {
    key: string;
    type: 'ESSAY' | 'DEGREE';
    prompt: string;
    rubric?: string;
    pointsMax: number;
    threshold?: number;
    scaleMin?: number;
    scaleMax?: number;
}

export interface CreateSurveyRequest {
    title: string;
    intent: string;
    settings: SurveySettings;
    questions: Omit<Question, 'key'>[];
}

export interface Room {
    id: string;
    roomCode: string;
    surveyId: string;
    status: 'waiting' | 'active' | 'ended';
    createdAt: string;
}

export interface CreateRoomResponse {
    roomCode: string;
}

export interface JoinRoomResponse {
    playerId: string;
    token: string;
    firstQuestion: Question;
}

export interface SubmitAnswerRequest {
    questionKey: string;
    textAnswer?: string;
    degreeValue?: number;
    clientAttemptId: string;
}

export interface SubmitAnswerResponse {
    status: 'EVALUATED';
    resolution: 'SAT' | 'UNSAT';
    pointsEarned: number;
    evalSummary: string;
    followUp: Question | null;
    nextQuestion: Question | null;
}

export interface RoomSnapshot {
    roomCode: string;
    endedAt: string;
    leaderboard: LeaderboardEntry[];
    completionRate: number;
    skipRate: number;
    questionProfiles: QuestionProfile[];
    roomSummary: RoomSummary;
}

export interface LeaderboardEntry {
    playerId: string;
    nickname: string;
    score: number;
    rank: number;
}

export interface QuestionProfile {
    key: string;
    prompt: string;
    type: string;
    satCount: number;
    unsatCount: number;
    skipCount: number;
    topThemes: string[];
    topMissing: string[];
    misunderstandings: string[];
    ratingHist?: number[];
    mean?: number;
    median?: number;
}

export interface RoomSummary {
    topThemes: string[];
    contrasts: Contrast[];
    frictionPoints: string[];
}

export interface Contrast {
    axis: string;
    sideA: string;
    sideB: string;
}

export interface AIReport {
    status: 'pending' | 'ready' | 'error';
    executiveSummary?: string[];
    keyThemes?: ThemeInsight[];
    contrasts?: ContrastInsight[];
    perQuestionInsights?: QuestionInsight[];
    frictionAnalysis?: FrictionPoint[];
    recommendedNextQuestions?: string[];
    recommendedEdits?: RecommendedEdit[];
}

export interface ThemeInsight {
    name: string;
    meaning: string;
    percentage: number;
    evidence: string[];
}

export interface ContrastInsight {
    axis: string;
    sideA: string;
    sideB: string;
    predictor?: string;
}

export interface QuestionInsight {
    questionKey: string;
    whatWorked: string;
    misunderstandings: string[];
    missingDetails: string[];
    effectiveFollowUps: string[];
}

export interface FrictionPoint {
    questionKey: string;
    skipRate: number;
    hypothesizedReason: string;
}

export interface RecommendedEdit {
    questionKey: string;
    original: string;
    suggested: string;
    reason: string;
}

// ============================================
// API Client
// ============================================

class ApiError extends Error {
    constructor(public status: number, message: string) {
        super(message);
        this.name = 'ApiError';
    }
}

async function request<T>(
    endpoint: string,
    options: RequestInit = {}
): Promise<T> {
    const url = `${API_BASE}${endpoint}`;

    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...((options.headers as Record<string, string>) || {}),
    };

    const response = await fetch(url, {
        ...options,
        headers,
    });

    const data = await response.json();

    if (!response.ok) {
        throw new ApiError(response.status, data.error || 'Request failed');
    }

    return data as T;
}

function getToken(type: 'host' | 'player'): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem(type === 'host' ? 'host_token' : 'player_token');
}

function authHeaders(type: 'host' | 'player'): Record<string, string> {
    const token = getToken(type);
    return token ? { Authorization: `Bearer ${token}` } : {};
}

// ============================================
// Auth API
// ============================================

export const auth = {
    login: async (username: string, password: string): Promise<LoginResponse> => {
        const response = await request<LoginResponse>('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
        });
        localStorage.setItem('host_token', response.token);
        return response;
    },

    logout: () => {
        localStorage.removeItem('host_token');
    },

    isLoggedIn: (): boolean => {
        return !!getToken('host');
    },
};

// ============================================
// Surveys API
// ============================================

export const surveys = {
    create: async (data: CreateSurveyRequest): Promise<Survey> => {
        return request<Survey>('/surveys', {
            method: 'POST',
            headers: authHeaders('host'),
            body: JSON.stringify(data),
        });
    },

    list: async (): Promise<Survey[]> => {
        return request<Survey[]>('/surveys', {
            headers: authHeaders('host'),
        });
    },

    get: async (id: string): Promise<Survey> => {
        return request<Survey>(`/surveys/${id}`, {
            headers: authHeaders('host'),
        });
    },
};

// ============================================
// Rooms API
// ============================================

export const rooms = {
    create: async (surveyId: string): Promise<CreateRoomResponse> => {
        return request<CreateRoomResponse>('/rooms', {
            method: 'POST',
            headers: authHeaders('host'),
            body: JSON.stringify({ surveyId }),
        });
    },

    start: async (code: string): Promise<void> => {
        await request(`/rooms/${code}/start`, {
            method: 'POST',
            headers: authHeaders('host'),
        });
    },

    end: async (code: string): Promise<void> => {
        await request(`/rooms/${code}/end`, {
            method: 'POST',
            headers: authHeaders('host'),
        });
    },

    join: async (code: string, nickname: string): Promise<JoinRoomResponse> => {
        const response = await request<JoinRoomResponse>(`/rooms/${code}/join`, {
            method: 'POST',
            body: JSON.stringify({ nickname }),
        });
        localStorage.setItem('player_token', response.token);
        localStorage.setItem('player_id', response.playerId);
        localStorage.setItem('room_code', code);
        return response;
    },
};

// ============================================
// Player API
// ============================================

export const player = {
    getCurrentQuestion: async (code: string): Promise<Question> => {
        return request<Question>(`/rooms/${code}/question/current`, {
            headers: authHeaders('player'),
        });
    },

    saveDraft: async (code: string, questionKey: string, draft: string): Promise<void> => {
        await request(`/rooms/${code}/questions/${questionKey}/draft`, {
            method: 'PUT',
            headers: authHeaders('player'),
            body: JSON.stringify({ draft }),
        });
    },

    submitAnswer: async (
        code: string,
        data: SubmitAnswerRequest
    ): Promise<SubmitAnswerResponse> => {
        return request<SubmitAnswerResponse>(`/rooms/${code}/answers`, {
            method: 'POST',
            headers: authHeaders('player'),
            body: JSON.stringify(data),
        });
    },

    skipQuestion: async (code: string, questionKey: string): Promise<void> => {
        await request(`/rooms/${code}/questions/${questionKey}/skip`, {
            method: 'POST',
            headers: authHeaders('player'),
        });
    },
};

// ============================================
// Reports API
// ============================================

export const reports = {
    getSnapshot: async (code: string): Promise<RoomSnapshot> => {
        return request<RoomSnapshot>(`/reports/${code}/snapshot`, {
            headers: authHeaders('host'),
        });
    },

    triggerAIReport: async (code: string): Promise<void> => {
        await request(`/reports/${code}/ai`, {
            method: 'POST',
            headers: authHeaders('host'),
        });
    },

    getAIReport: async (code: string): Promise<AIReport> => {
        return request<AIReport>(`/reports/${code}/ai`, {
            headers: authHeaders('host'),
        });
    },
};

// ============================================
// WebSocket URLs
// ============================================

export function getHostWebSocketUrl(code: string): string {
    const token = getToken('host');
    const wsBase = API_BASE.replace('http', 'ws');
    return `${wsBase}/ws/rooms/${code}/host?token=${token}`;
}

export function getPlayerWebSocketUrl(code: string): string {
    const token = getToken('player');
    const wsBase = API_BASE.replace('http', 'ws');
    return `${wsBase}/ws/rooms/${code}/player?token=${token}`;
}
