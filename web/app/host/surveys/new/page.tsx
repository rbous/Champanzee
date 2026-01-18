'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { surveys } from '@/lib/api';
import SurveyForm, { QuestionInput } from '@/components/SurveyForm';

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
                    threshold: 0.6,
                    scaleMin: 1,
                    scaleMax: 5,
                },
                {
                    type: 'ESSAY',
                    prompt: "Which feature do you find the most impressive? (Display, Battery, Camera, Speed, Design)",
                    rubric: "Look for specific mention of one of the listed features and why they like it.",
                    pointsMax: 100,
                    threshold: 0.6,
                    scaleMin: 1,
                    scaleMax: 5,
                },
                {
                    type: 'DEGREE',
                    prompt: "How would you rate the phone’s performance during everyday tasks?",
                    rubric: "",
                    pointsMax: 50,
                    threshold: 0.6,
                    scaleMin: 1,
                    scaleMax: 5,
                },
                {
                    type: 'ESSAY',
                    prompt: "What was the main reason you chose this phone? (Price, Features, Brand, Design, Reviews)",
                    rubric: "Identify the primary motivation factor.",
                    pointsMax: 100,
                    threshold: 0.6,
                    scaleMin: 1,
                    scaleMax: 5,
                },
                {
                    type: 'ESSAY',
                    prompt: "What is one thing you would improve or change about this smartphone?",
                    rubric: "Constructive criticism or specific feature requests.",
                    pointsMax: 100,
                    threshold: 0.6,
                    scaleMin: 1,
                    scaleMax: 5,
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

    return (
        <div className="min-h-screen p-6 max-w-4xl mx-auto">
            {/* Header */}
            <div className="flex items-center justify-between mb-8">
                <div className="flex items-center gap-4">
                    <a href="/host" className="btn btn-ghost">← Back</a>
                    <div>
                        <h1 className="text-2xl font-bold">Create Survey</h1>
                        <p className="text-[var(--foreground-muted)] text-sm">
                            Design your AI-powered survey questions
                        </p>
                    </div>
                </div>
                <button
                    onClick={loadTemplate}
                    className="btn btn-secondary"
                >
                    📱 Load Smartphone Template
                </button>
            </div>

            <SurveyForm
                initialTitle={formData.title}
                initialIntent={formData.intent}
                initialQuestions={formData.questions.length > 0 ? formData.questions : undefined}
                onSubmit={handleSubmit}
                submitLabel="Create Survey"
                isLoading={loading}
                error={error}
            />
        </div>
    );
}
