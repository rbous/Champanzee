'use client';

import { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { surveys } from '@/lib/api';
import SurveyForm, { QuestionInput, toQuestionInput } from '@/components/SurveyForm';

export default function EditSurvey() {
    const router = useRouter();
    const params = useParams();
    const id = params?.id as string;

    const [loading, setLoading] = useState(true);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState('');

    // Initial data state
    const [initialTitle, setInitialTitle] = useState('');
    const [initialIntent, setInitialIntent] = useState('');
    const [initialMaxFollowUps, setInitialMaxFollowUps] = useState(2);
    const [initialAllowSkipAfter, setInitialAllowSkipAfter] = useState(1);
    const [initialQuestions, setInitialQuestions] = useState<QuestionInput[]>([]);

    useEffect(() => {
        if (id) {
            loadSurvey(id);
        }
    }, [id]);

    const loadSurvey = async (surveyId: string) => {
        try {
            const survey = await surveys.get(surveyId);
            setInitialTitle(survey.title);
            setInitialIntent(survey.intent);
            setInitialMaxFollowUps(survey.settings.maxFollowUps);
            setInitialAllowSkipAfter(survey.settings.allowSkipAfter);
            setInitialQuestions(survey.questions.map(toQuestionInput));
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load survey');
        } finally {
            setLoading(false);
        }
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
        setSubmitting(true);
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

            await surveys.update(id, {
                title: data.title,
                intent: data.intent,
                settings: data.settings,
                questions: formattedQuestions,
            });

            router.push('/host');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update survey');
        } finally {
            setSubmitting(false);
        }
    };

    if (loading) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="spinner" style={{ width: 40, height: 40 }} />
            </div>
        );
    }

    return (
        <div className="min-h-screen p-6 max-w-4xl mx-auto">
            {/* Header */}
            <div className="flex items-center gap-4 mb-8">
                <a href="/host" className="btn btn-ghost">← Back</a>
                <div>
                    <h1 className="text-2xl font-bold">Edit Survey</h1>
                    <p className="text-[var(--foreground-muted)] text-sm">
                        Update your survey questions and settings
                    </p>
                </div>
            </div>

            <SurveyForm
                initialTitle={initialTitle}
                initialIntent={initialIntent}
                initialMaxFollowUps={initialMaxFollowUps}
                initialAllowSkipAfter={initialAllowSkipAfter}
                initialQuestions={initialQuestions}
                onSubmit={handleSubmit}
                submitLabel="Update Survey"
                isLoading={submitting}
                error={error}
            />
        </div>
    );
}
