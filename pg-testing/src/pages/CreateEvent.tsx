import React, { useState } from 'react';
import { Copy, Link, Calendar, Banknote } from 'lucide-react';

const CreateEvent = () => {
    const [eventName, setEventName] = useState('');
    const [amount, setAmount] = useState('');
    const [generatedLink, setGeneratedLink] = useState('');

    const handleCreate = (e: React.FormEvent) => {
        e.preventDefault();
        if (!eventName || !amount) return;

        // Use URLSearchParams to ensure proper encoding
        const params = new URLSearchParams({
            event: eventName,
            amount: amount
        });

        // Construct the full URL
        const url = `${window.location.origin}/register?${params.toString()}`;
        setGeneratedLink(url);
    };

    const copyToClipboard = () => {
        navigator.clipboard.writeText(generatedLink);
        alert('Link copied to clipboard!');
    };

    return (
        <div className="flex flex-col items-center justify-center min-h-screen p-4 bg-gray-900 text-white">
            <div className="w-full max-w-md p-8 bg-gray-800 rounded-2xl shadow-xl border border-gray-700">
                <h1 className="text-3xl font-bold mb-2 text-center text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-emerald-400">
                    Create Event
                </h1>
                <p className="text-gray-400 text-center mb-8">Generate a payment link for your event</p>

                <form onSubmit={handleCreate} className="space-y-6">
                    <div className="space-y-2">
                        <label className="text-sm font-medium text-gray-300 flex items-center gap-2">
                            <Calendar className="w-4 h-4" /> Event Name
                        </label>
                        <input
                            type="text"
                            value={eventName}
                            onChange={(e) => setEventName(e.target.value)}
                            placeholder="e.g. Summer Hackathon"
                            className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none transition-all placeholder-gray-500 text-white"
                        />
                    </div>

                    <div className="space-y-2">
                        <label className="text-sm font-medium text-gray-300 flex items-center gap-2">
                            <Banknote className="w-4 h-4" /> Ticket Amount (₹)
                        </label>
                        <input
                            type="number"
                            value={amount}
                            onChange={(e) => setAmount(e.target.value)}
                            placeholder="e.g. 500"
                            className="w-full px-4 py-3 bg-gray-700 border border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none transition-all placeholder-gray-500 text-white"
                        />
                    </div>

                    <button
                        type="submit"
                        className="w-full py-3 px-4 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white font-bold rounded-lg shadow-lg hover:shadow-blue-500/25 transition-all transform hover:-translate-y-0.5"
                    >
                        Generate Link
                    </button>
                </form>

                {generatedLink && (
                    <div className="mt-8 p-4 bg-gray-700/50 rounded-lg border border-gray-600 animate-fade-in">
                        <label className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2 block">
                            Share this link
                        </label>
                        <div className="flex items-center gap-2 bg-gray-800 p-2 rounded border border-gray-700">
                            <Link className="w-4 h-4 text-gray-500 flex-shrink-0" />
                            <input
                                readOnly
                                value={generatedLink}
                                className="bg-transparent text-sm text-gray-300 w-full outline-none"
                            />
                            <button
                                onClick={copyToClipboard}
                                className="p-2 hover:bg-gray-700 rounded-md text-blue-400 transition-colors"
                                title="Copy to clipboard"
                            >
                                <Copy className="w-5 h-5" />
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};

export default CreateEvent;
