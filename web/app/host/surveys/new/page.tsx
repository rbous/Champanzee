'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { surveys } from '@/lib/api';
import SurveyForm, { QuestionInput } from '@/components/SurveyForm';
import GameBackground from '@/components/GameBackground';

export default function NewSurvey() {
    const router = useRouter();
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const [formData, setFormData] = useState<{
        title: string;
        intent: string;
        questions: QuestionInput[];
    }>({
        title: '',
        intent: '',
        questions: [],
    });

    const loadTemplate = () => {
        setFormData({
            title: "Smartphone Launch Feedback",
            intent: "Understand user perception, satisfaction, and improvement areas for the new device.",
            questions: [
                {
                    type: 'DEGREE',
                    prompt: "On a scale from 1 to 5, how satisfied are you with this smartphone overall?",
                    rubric: "",
                    pointsMax: 50,
                    threshold: 0.5,
                    scaleMin: 1,
                    scaleMax: 5,
                    options: [],
                },
                {
                    type: 'MCQ',
                    prompt: "Which model did you purchase?",
                    rubric: "",
                    pointsMax: 50,
                    threshold: 0.5,
                    scaleMin: 1,
                    scaleMax: 5,
                    options: [
                        "Standard Model",
                        "Pro / Plus Model",
                        "Ultra / Max Model"
                    ],
                },
                {
                    type: 'ESSAY',
                    prompt: "Which feature do you find the most impressive? (Display, Battery, Camera, Speed, Design)",
                    rubric: "Look for specific mention of one of the listed features and why they like it.",
                    pointsMax: 100,
                    threshold: 0.5,
                    scaleMin: 1,
                    scaleMax: 5,
                    options: [],
                },
                {
                    type: 'DEGREE',
                    prompt: "How would you rate the phone's performance during everyday tasks?",
                    rubric: "",
                    pointsMax: 50,
                    threshold: 0.5,
                    scaleMin: 1,
                    scaleMax: 5,
                    options: [],
                },
                {
                    type: 'ESSAY',
                    prompt: "What was the main reason you chose this phone? (Price, Features, Brand, Design, Reviews)",
                    rubric: "Identify the primary motivation factor.",
                    pointsMax: 100,
                    threshold: 0.5,
                    scaleMin: 1,
                    scaleMax: 5,
                    options: [],
                },
                {
                    type: 'ESSAY',
                    prompt: "What is one thing you would improve or change about this smartphone?",
                    rubric: "Constructive criticism or specific feature requests.",
                    pointsMax: 100,
                    threshold: 0.5,
                    scaleMin: 1,
                    scaleMax: 5,
                    options: [],
                }
            ]
        });
    };

    const handleSubmit = async (data: {
        title: string;
        intent: string;
        settings: {
            maxFollowUps: number;
            allowSkipAfter: number;
        };
        questions: QuestionInput[];
    }) => {
        setLoading(true);
        setError('');

        try {
            const formattedQuestions = data.questions.map((q) => ({
                type: q.type,
                prompt: q.prompt,
                pointsMax: q.pointsMax,
                ...(q.type === 'ESSAY' && {
                    rubric: q.rubric,
                    threshold: q.threshold,
                }),
                ...(q.type === 'DEGREE' && {
                    scaleMin: q.scaleMin,
                    scaleMax: q.scaleMax,
                }),
                ...(q.type === 'MCQ' && {
                    options: q.options.filter(opt => opt.trim() !== ''), // Filter out empty options
                }),
            }));

            await surveys.create({
                title: data.title,
                intent: data.intent,
                settings: data.settings,
                questions: formattedQuestions,
            });

            router.push('/host');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to create survey');
        } finally {
            setLoading(false);
        }
    };

    const generateFromInsights = async () => {
        if (!formData.intent) {
            setError("Please enter a specific Intent above so AI can find relevant past questions.");
            return;
        }
        setLoading(true);
        setError('');

        try {
            const newQuestions = await surveys.generateFromInsights(formData.intent);
            if (newQuestions.length === 0) {
                setError("No relevant past insights found for this intent.");
            } else {
                setFormData(prev => ({
                    ...prev,
                    questions: [...prev.questions, ...newQuestions.map(q => ({
                        type: q.type,
                        prompt: q.prompt,
                        rubric: q.rubric || '',
                        pointsMax: q.pointsMax || 50,
                        threshold: q.threshold || 0.6,
                        scaleMin: q.scaleMin || 1,
                        scaleMax: q.scaleMax || 5,
                        options: q.options || [],
                    } as QuestionInput))]
                }));
            }
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to generate from insights');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen p-6 relative">
            <GameBackground />

            <div className="relative z-10 max-w-4xl mx-auto">
                {/* Header */}
                <div className="flex items-center justify-between mb-8">
                    <div className="flex items-center gap-4">
                        <a href="/host" className="btn btn-secondary rotate-1 hover:-rotate-1">
                            ← Back
                        </a>
                        <div>
                            <h1 className="text-3xl font-black">
                                <span className="text-party-gradient">✨ Create Survey</span>
                            </h1>
                            <p className="text-[var(--text-muted)] font-bold">
                                Design your AI-powered survey questions
                            </p>
                        </div>
                    </div>
                    <div className="flex gap-2">
                        <button
                            onClick={generateFromInsights}
                            className={`btn bg-purple-600 text-white hover:bg-purple-700 hover:scale-105 ${loading ? 'opacity-50 cursor-not-allowed' : ''}`}
                            disabled={loading}
                        >
                            ✨ Generate from Insights
                        </button>
                        <button
                            onClick={loadTemplate}
                            className="btn btn-yellow hover:scale-105 hover:rotate-2"
                        >
                            📱 Load Template
                        </button>
                    </div>
                </div>

                <SurveyForm
                    initialTitle={formData.title}
                    initialIntent={formData.intent}
                    initialQuestions={formData.questions.length > 0 ? formData.questions : undefined}
                    onSubmit={handleSubmit}
                    submitLabel="🚀 Create Survey"
                    isLoading={loading}
                    error={error}
                />
            </div>
        </div>
    );
}
