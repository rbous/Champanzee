const getApiBase = () => {
    if (typeof window !== 'undefined' && (window as any).__RUNTIME_CONFIG__?.API_URL) {
        return (window as any).__RUNTIME_CONFIG__.API_URL;
    }
    return process.env.NEXT_PUBLIC_API_URL || 'http://api.champanzee.tech/v1';
};

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
    type: 'ESSAY' | 'DEGREE' | 'MCQ';
    prompt: string;
    rubric?: string;
    pointsMax: number;
    threshold?: number;
    scaleMin?: number;
    scaleMax?: number;
    options?: string[]; // MCQ only
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
    roomMeta: {
        surveyId: string;
        hostId: string;
        status: string;
        createdAt: string;
        settingsJson: string;
        scopeSummary: string;
    };
    firstQuestion: Question;
}

export interface SubmitAnswerRequest {
    questionKey: string;
    textAnswer?: string;
    degreeValue?: number;
    optionIndex?: number;
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
    surveyId: string;
    smSurveyId?: string;
    smWebLink?: string;
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
    ratingSum: number;
    ratingCount: number;
    optionHist: { [key: number]: number };
    answerCount: number;
    topThemes: string[];
    topMissing: string[];
    misunderstandings: string[];
    ratingHist?: number[];
    mean?: number;
    median?: number;
    satCount?: number;
    unsatCount?: number;
    skipCount?: number;
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

export interface SkipQuestionResponse {
    done: boolean;
    nextQuestion: Question | null;
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
    const API_BASE = getApiBase();
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
        const response = await request<{ surveys: Survey[] }>('/surveys', {
            headers: authHeaders('host'),
        });
        return response.surveys;
    },

    update: async (id: string, data: CreateSurveyRequest): Promise<Survey> => {
        return request<Survey>(`/surveys/${id}`, {
            method: 'PUT',
            headers: authHeaders('host'),
            body: JSON.stringify(data),
        });
    },

    get: async (id: string): Promise<Survey> => {
        return request<Survey>(`/surveys/${id}`, {
            headers: authHeaders('host'),
        });
    },

    generateFromInsights: async (intent: string): Promise<Question[]> => {
        const response = await request<{ questions: Question[] }>('/surveys/generate-from-insights', {
            method: 'POST',
            headers: authHeaders('host'),
            body: JSON.stringify({ intent }),
        });
        return response.questions;
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

        // Store room settings if available
        if (response.roomMeta && response.roomMeta.settingsJson) {
            try {
                const settings = JSON.parse(response.roomMeta.settingsJson);
                // The Go model has AllowSkipAfter (int)
                if (settings.allowSkipAfter !== undefined) {
                    localStorage.setItem('room_allow_skip_after', String(settings.allowSkipAfter));
                } else if (settings.allowSkipImmediately !== undefined) {
                    // Fallback for legacy rooms/settings
                    localStorage.setItem('room_allow_skip_after', settings.allowSkipImmediately ? '0' : '1');
                }
            } catch (e) {
                console.error("Failed to parse room settings", e);
            }
        }

        return response;
    },
};

// ============================================
// Player API
// ============================================

export const player = {
    getCurrentQuestion: async (code: string): Promise<{ question: Question | null, player?: { score: number } }> => {
        const response = await request<{ question: Question | null, player?: { score: number } }>(`/rooms/${code}/question/current`, {
            headers: authHeaders('player'),
        });
        return response;
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

    skipQuestion: async (code: string, questionKey: string): Promise<SkipQuestionResponse> => {
        return request<SkipQuestionResponse>(`/rooms/${code}/questions/${questionKey}/skip`, {
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
// SurveyMonkey API
// ============================================

export interface SMSurveyResponse {
    smSurveyId: string;
    weblinkUrl: string;
    title: string;
}

export interface SMSyncResult {
    fetched: number;
    insertedRaw: number;
    parsedAnswers: number;
    updatedFeatures: number;
}

export interface SMSummary {
    totalResponses: number;
    avgOverallSatisfaction: number;
    topFeatureCounts: Array<{ feature: string; count: number }>;
    latestSubmittedAt?: string;
}

export const surveymonkey = {
    createFromInternal: async (surveyId: string, recommendedNextQuestions?: string[]): Promise<SMSurveyResponse> => {
        return request<SMSurveyResponse>('/sm/surveys/from-internal', {
            method: 'POST',
            headers: authHeaders('host'),
            body: JSON.stringify({ surveyId, recommendedNextQuestions }),
        });
    },

    sync: async (smSurveyId: string): Promise<SMSyncResult> => {
        return request<SMSyncResult>(`/sm/surveys/${smSurveyId}/sync`, {
            method: 'POST',
            headers: authHeaders('host'),
        });
    },

    getSummary: async (smSurveyId: string): Promise<SMSummary> => {
        return request<SMSummary>(`/sm/surveys/${smSurveyId}/summary`, {
            headers: authHeaders('host'),
        });
    },
};

// ============================================
// WebSocket URLs
// ============================================

export function getHostWebSocketUrl(code: string): string | null {
    const token = getToken('host');
    if (!token) return null;
    const API_BASE = getApiBase();
    const wsBase = API_BASE.replace('http', 'ws');
    return `${wsBase}/ws/rooms/${code}/host?token=${token}`;
}

export function getPlayerWebSocketUrl(code: string): string | null {
    const token = getToken('player');
    if (!token) return null;
    const API_BASE = getApiBase();
    const wsBase = API_BASE.replace('http', 'ws');
    return `${wsBase}/ws/rooms/${code}/player?token=${token}`;
}
