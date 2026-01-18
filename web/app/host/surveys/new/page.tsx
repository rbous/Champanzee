'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { surveys, type Question } from '@/lib/api';

type QuestionInput = {
    type: 'ESSAY' | 'DEGREE';
    prompt: string;
    rubric: string;
    pointsMax: number;
    threshold: number;
    scaleMin: number;
    scaleMax: number;
};

const defaultQuestion: QuestionInput = {
    type: 'ESSAY',
    prompt: '',
    rubric: '',
    pointsMax: 100,
    threshold: 0.6,
    scaleMin: 1,
    scaleMax: 5,
};

export default function NewSurvey() {
    const router = useRouter();
    const [title, setTitle] = useState('');
    const [intent, setIntent] = useState('');
    const [maxFollowUps, setMaxFollowUps] = useState(2);
    const [allowSkipAfter, setAllowSkipAfter] = useState(1);
    const [questions, setQuestions] = useState<QuestionInput[]>([{ ...defaultQuestion }]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const addQuestion = () => {
        setQuestions([...questions, { ...defaultQuestion }]);
    };

    const removeQuestion = (index: number) => {
        if (questions.length > 1) {
            setQuestions(questions.filter((_, i) => i !== index));
        }
    };

    const updateQuestion = (index: number, field: keyof QuestionInput, value: unknown) => {
        const updated = [...questions];
        updated[index] = { ...updated[index], [field]: value };
        setQuestions(updated);
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!title.trim() || !intent.trim()) {
            setError('Please fill in title and intent');
            return;
        }

        if (questions.some(q => !q.prompt.trim())) {
            setError('All questions must have a prompt');
            return;
        }

        setLoading(true);
        setError('');

        try {
            const formattedQuestions = questions.map((q) => ({
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
                title,
                intent,
                settings: { maxFollowUps, allowSkipAfter },
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
            <div className="flex items-center gap-4 mb-8">
                <a href="/host" className="btn btn-ghost">← Back</a>
                <div>
                    <h1 className="text-2xl font-bold">Create Survey</h1>
                    <p className="text-[var(--foreground-muted)] text-sm">
                        Design your AI-powered survey questions
                    </p>
                </div>
            </div>

            <form onSubmit={handleSubmit} className="space-y-8">
                {/* Basic Info */}
                <div className="card">
                    <h2 className="text-lg font-semibold mb-4">Survey Details</h2>

                    <div className="space-y-4">
                        <div>
                            <label className="input-label">Title</label>
                            <input
                                type="text"
                                className="input"
                                placeholder="e.g., Product Feedback Survey"
                                value={title}
                                onChange={(e) => setTitle(e.target.value)}
                            />
                        </div>

                        <div>
                            <label className="input-label">Intent / Goal</label>
                            <textarea
                                className="input"
                                placeholder="What do you want to learn from this survey?"
                                value={intent}
                                onChange={(e) => setIntent(e.target.value)}
                                rows={2}
                            />
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <label className="input-label">Max Follow-ups per Question</label>
                                <input
                                    type="number"
                                    className="input"
                                    min={0}
                                    max={5}
                                    value={maxFollowUps}
                                    onChange={(e) => setMaxFollowUps(parseInt(e.target.value) || 0)}
                                />
                            </div>
                            <div>
                                <label className="input-label">Allow Skip After (attempts)</label>
                                <input
                                    type="number"
                                    className="input"
                                    min={0}
                                    max={5}
                                    value={allowSkipAfter}
                                    onChange={(e) => setAllowSkipAfter(parseInt(e.target.value) || 0)}
                                />
                            </div>
                        </div>
                    </div>
                </div>

                {/* Questions */}
                <div className="space-y-4">
                    <div className="flex items-center justify-between">
                        <h2 className="text-lg font-semibold">Questions</h2>
                        <button type="button" onClick={addQuestion} className="btn btn-secondary">
                            + Add Question
                        </button>
                    </div>

                    {questions.map((q, index) => (
                        <div key={index} className="card">
                            <div className="flex items-center justify-between mb-4">
                                <span className="badge badge-neutral">Q{index + 1}</span>
                                {questions.length > 1 && (
                                    <button
                                        type="button"
                                        onClick={() => removeQuestion(index)}
                                        className="btn btn-ghost text-[var(--error)] text-sm"
                                    >
                                        Remove
                                    </button>
                                )}
                            </div>

                            <div className="space-y-4">
                                {/* Question Type */}
                                <div>
                                    <label className="input-label">Type</label>
                                    <div className="flex gap-2">
                                        <button
                                            type="button"
                                            onClick={() => updateQuestion(index, 'type', 'ESSAY')}
                                            className={`btn flex-1 ${q.type === 'ESSAY' ? 'btn-primary' : 'btn-secondary'}`}
                                        >
                                            📝 Essay
                                        </button>
                                        <button
                                            type="button"
                                            onClick={() => updateQuestion(index, 'type', 'DEGREE')}
                                            className={`btn flex-1 ${q.type === 'DEGREE' ? 'btn-primary' : 'btn-secondary'}`}
                                        >
                                            📊 Rating
                                        </button>
                                    </div>
                                </div>

                                {/* Prompt */}
                                <div>
                                    <label className="input-label">Question Prompt</label>
                                    <textarea
                                        className="input"
                                        placeholder="What would you like to ask?"
                                        value={q.prompt}
                                        onChange={(e) => updateQuestion(index, 'prompt', e.target.value)}
                                        rows={2}
                                    />
                                </div>

                                {/* Essay-specific fields */}
                                {q.type === 'ESSAY' && (
                                    <>
                                        <div>
                                            <label className="input-label">Rubric (AI evaluation criteria)</label>
                                            <textarea
                                                className="input"
                                                placeholder="What makes a good answer? e.g., Include specific examples, mention tradeoffs..."
                                                value={q.rubric}
                                                onChange={(e) => updateQuestion(index, 'rubric', e.target.value)}
                                                rows={2}
                                            />
                                        </div>
                                        <div className="grid grid-cols-2 gap-4">
                                            <div>
                                                <label className="input-label">Max Points</label>
                                                <input
                                                    type="number"
                                                    className="input"
                                                    min={1}
                                                    max={1000}
                                                    value={q.pointsMax}
                                                    onChange={(e) => updateQuestion(index, 'pointsMax', parseInt(e.target.value) || 100)}
                                                />
                                            </div>
                                            <div>
                                                <label className="input-label">SAT Threshold (0-1)</label>
                                                <input
                                                    type="number"
                                                    className="input"
                                                    min={0}
                                                    max={1}
                                                    step={0.1}
                                                    value={q.threshold}
                                                    onChange={(e) => updateQuestion(index, 'threshold', parseFloat(e.target.value) || 0.6)}
                                                />
                                            </div>
                                        </div>
                                    </>
                                )}

                                {/* Degree-specific fields */}
                                {q.type === 'DEGREE' && (
                                    <div className="grid grid-cols-3 gap-4">
                                        <div>
                                            <label className="input-label">Min Scale</label>
                                            <input
                                                type="number"
                                                className="input"
                                                min={1}
                                                max={10}
                                                value={q.scaleMin}
                                                onChange={(e) => updateQuestion(index, 'scaleMin', parseInt(e.target.value) || 1)}
                                            />
                                        </div>
                                        <div>
                                            <label className="input-label">Max Scale</label>
                                            <input
                                                type="number"
                                                className="input"
                                                min={1}
                                                max={10}
                                                value={q.scaleMax}
                                                onChange={(e) => updateQuestion(index, 'scaleMax', parseInt(e.target.value) || 5)}
                                            />
                                        </div>
                                        <div>
                                            <label className="input-label">Points</label>
                                            <input
                                                type="number"
                                                className="input"
                                                min={1}
                                                max={100}
                                                value={q.pointsMax}
                                                onChange={(e) => updateQuestion(index, 'pointsMax', parseInt(e.target.value) || 20)}
                                            />
                                        </div>
                                    </div>
                                )}
                            </div>
                        </div>
                    ))}
                </div>

                {/* Error */}
                {error && (
                    <div className="text-[var(--error)] text-center animate-fade-in">
                        {error}
                    </div>
                )}

                {/* Submit */}
                <button
                    type="submit"
                    disabled={loading}
                    className="btn btn-primary w-full py-4 text-lg"
                >
                    {loading ? (
                        <span className="flex items-center gap-2">
                            <div className="spinner" style={{ width: 20, height: 20 }} />
                            Creating...
                        </span>
                    ) : (
                        'Create Survey'
                    )}
                </button>
            </form>
        </div>
    );
}
