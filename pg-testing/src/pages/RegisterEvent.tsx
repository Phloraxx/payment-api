import { useEffect, useState } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import axios from 'axios';
import { QRCodeSVG } from 'qrcode.react';
import { motion, AnimatePresence } from 'framer-motion';
import {
    CircleNotch,
    CheckCircle,
    WarningCircle,
    DownloadSimple,
    ShieldCheck,
    Lightning,
    QrCode,
    ArrowRight,
    CreditCard
} from '@phosphor-icons/react';

const API_URL = 'https://payment-api.nerdpixel.workers.dev/api';

interface TicketResponse {
    id: string;
    amount: number;
    status: string;
}

const RegisterEvent = () => {
    const [searchParams, setSearchParams] = useSearchParams();
    const navigate = useNavigate();

    const initialAmount = searchParams.get('amount') || '';
    const ticketId = searchParams.get('ticketId');

    const [amount, setAmount] = useState(initialAmount);

    const [status, setStatus] = useState<'idle' | 'loading' | 'pending' | 'paid'>(
        ticketId ? 'pending' : 'idle'
    );
    const [error, setError] = useState('');

    const urlAmount = searchParams.get('amount');
    const urlTicketId = searchParams.get('ticketId');
    const [prevUrlAmount, setPrevUrlAmount] = useState(urlAmount);

    if (urlAmount !== prevUrlAmount) {
        setPrevUrlAmount(urlAmount);
        if (urlAmount) {
            setAmount(urlAmount);
        }
    }

    if (urlTicketId && status === 'idle') {
        setStatus('pending');
    }

    const createTicket = async () => {
        if (!amount || isNaN(Number(amount)) || Number(amount) <= 0) {
            setError('Please enter a valid amount');
            return;
        }

        try {
            setStatus('loading');
            setError('');
            const res = await axios.post<TicketResponse>(`${API_URL}/ticket`, {
                amount: Number(amount)
            });
            console.log('Ticket Created:', res.data);

            setSearchParams(prev => {
                const newParams = new URLSearchParams(prev);
                newParams.set('ticketId', res.data.id);
                newParams.set('amount', amount);
                return newParams;
            });

            setStatus('pending');
        } catch (err) {
            console.error(err);
            setError('System error. Please try generating the code again.');
            setStatus('idle');
        }
    };

    useEffect(() => {
        let interval: ReturnType<typeof setInterval>;

        if (ticketId && status === 'pending') {
            interval = setInterval(async () => {
                try {
                    const res = await axios.get<TicketResponse>(`${API_URL}/status/${ticketId}`);

                    if (res.data.status === 'paid') {
                        setStatus('paid');
                        clearInterval(interval);
                        setTimeout(() => {
                            navigate(`/ticket/${ticketId}`);
                        }, 1500);
                    }
                } catch (err) {
                    console.error('Polling error', err);
                }
            }, 2000);
        }

        return () => clearInterval(interval);
    }, [ticketId, status, navigate]);

    const upiUrl = ticketId
        ? `upi://pay?pa=souravpbijoy-2@okicici&am=${amount}&tn=${ticketId}&cu=INR&pn=IEEESahrdaya`
        : '';

    const downloadQR = () => {
        const svg = document.querySelector('#qr-code-container svg');
        if (svg) {
            const svgData = new XMLSerializer().serializeToString(svg);
            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');
            const img = new Image();

            const padding = 24;

            img.onload = () => {
                canvas.width = img.width + (padding * 2);
                canvas.height = img.height + (padding * 2);

                if (ctx) {
                    ctx.fillStyle = '#FFFFFF';
                    ctx.fillRect(0, 0, canvas.width, canvas.height);
                    ctx.drawImage(img, padding, padding);

                    const pngFile = canvas.toDataURL('image/png');
                    const downloadLink = document.createElement('a');
                    downloadLink.download = `ticket-${ticketId}.png`;
                    downloadLink.href = pngFile;
                    downloadLink.click();
                }
            };

            img.src = 'data:image/svg+xml;base64,' + btoa(unescape(encodeURIComponent(svgData)));
        }
    };

    return (
        <div className="min-h-[100dvh] bg-[#f9fafb] text-slate-900 font-sans p-6 md:p-12 flex flex-col justify-center overflow-x-hidden">
            <div className="max-w-[1200px] mx-auto w-full grid grid-cols-1 lg:grid-cols-12 gap-12 lg:gap-20 items-center">

                {/* Left Side: Asymmetrical Hero */}
                <div className="lg:col-span-7 flex flex-col justify-center pt-10 lg:pt-0">
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
                        className="space-y-8"
                    >
                        <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-slate-900/5 border border-slate-900/10">
                            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                            <span className="text-xs font-semibold tracking-wide text-slate-700 uppercase">Live Registration</span>
                        </div>

                        <h1 className="text-5xl lg:text-7xl tracking-tighter leading-[0.9] font-bold text-slate-950">
                            Sahrdaya IEEE<br />
                            <span className="text-slate-400">Event Portal.</span>
                        </h1>
                        <p className="text-lg text-slate-600 leading-relaxed max-w-[50ch] mt-6">
                            Secure your access instantly. Generate a unique cryptographic pass, scan via any UPI app, and receive your verified entry ticket.
                        </p>
                    </motion.div>

                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        transition={{ delay: 0.3, duration: 0.8 }}
                        className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-12"
                    >
                        <div className="bg-white p-8 rounded-[2rem] border border-slate-200/50 shadow-[0_20px_40px_-15px_rgba(0,0,0,0.03)] hover:-translate-y-1 transition-transform duration-300">
                            <ShieldCheck weight="duotone" className="w-8 h-8 text-slate-900 mb-5" />
                            <h3 className="text-lg font-semibold text-slate-900 tracking-tight">Verified Protocol</h3>
                            <p className="text-sm text-slate-500 mt-2 leading-relaxed">End-to-end secure transactions directly linked to your digital identity.</p>
                        </div>
                        <div className="bg-white p-8 rounded-[2rem] border border-slate-200/50 shadow-[0_20px_40px_-15px_rgba(0,0,0,0.03)] hover:-translate-y-1 transition-transform duration-300">
                            <Lightning weight="duotone" className="w-8 h-8 text-slate-900 mb-5" />
                            <h3 className="text-lg font-semibold text-slate-900 tracking-tight">Instant Issue</h3>
                            <p className="text-sm text-slate-500 mt-2 leading-relaxed">Zero latency ticket generation immediately post-payment verification.</p>
                        </div>
                    </motion.div>
                </div>

                {/* Right Side: Interactive Bento Container */}
                <div className="lg:col-span-5 relative w-full max-w-md mx-auto lg:mx-0">
                    {/* Background blob for depth */}
                    <div className="absolute inset-0 bg-slate-200/40 rounded-[3rem] blur-2xl -z-10 transform rotate-3 scale-105"></div>

                    <motion.div
                        layout
                        transition={{ type: "spring", stiffness: 100, damping: 20 }}
                        className="bg-white p-8 md:p-10 rounded-[2.5rem] border border-slate-200/80 shadow-[0_30px_60px_-15px_rgba(0,0,0,0.08)] relative overflow-hidden"
                    >
                        <div className="absolute top-0 right-0 p-8 opacity-5">
                            <QrCode weight="fill" className="w-32 h-32" />
                        </div>

                        <AnimatePresence mode="wait">
                            {status === 'idle' && (
                                <motion.div
                                    key="idle"
                                    initial={{ opacity: 0, x: 20 }}
                                    animate={{ opacity: 1, x: 0 }}
                                    exit={{ opacity: 0, x: -20 }}
                                    className="relative z-10"
                                >
                                    <div className="mb-8">
                                        <h2 className="text-2xl font-bold tracking-tight text-slate-900">Initialize Ticket</h2>
                                        <p className="text-sm text-slate-500 mt-1">Enter desired amount for the transaction.</p>
                                    </div>

                                    <div className="space-y-6">
                                        <div className="relative group">
                                            <label className="block text-slate-500 text-xs font-semibold uppercase tracking-wider mb-2">
                                                Amount
                                            </label>
                                            <div className="relative flex items-center">
                                                <span className="absolute left-4 text-slate-400 font-medium text-xl">₹</span>
                                                <input
                                                    type="number"
                                                    value={amount}
                                                    onChange={(e) => setAmount(e.target.value)}
                                                    placeholder="0.00"
                                                    className="w-full bg-slate-50/50 border border-slate-200 rounded-[1.25rem] py-4 pl-10 pr-4 text-2xl font-bold text-slate-900 placeholder-slate-300 focus:outline-none focus:ring-2 focus:ring-slate-900 focus:border-transparent transition-all"
                                                    autoFocus
                                                />
                                            </div>
                                            {error && (
                                                <motion.p
                                                    initial={{ opacity: 0, y: -10 }}
                                                    animate={{ opacity: 1, y: 0 }}
                                                    className="text-red-500 text-sm mt-3 flex items-center gap-1.5"
                                                >
                                                    <WarningCircle weight="fill" />
                                                    {error}
                                                </motion.p>
                                            )}
                                        </div>

                                        <button
                                            onClick={createTicket}
                                            disabled={!amount || Number(amount) <= 0}
                                            className="group relative w-full flex items-center justify-between bg-slate-950 text-white rounded-[1.25rem] py-4 px-6 font-semibold overflow-hidden disabled:opacity-50 disabled:cursor-not-allowed transition-all active:scale-[0.98]"
                                        >
                                            <span className="relative z-10">Generate Pass</span>
                                            <span className="relative z-10 flex items-center justify-center w-8 h-8 rounded-full bg-white/10 group-hover:bg-white/20 transition-colors">
                                                <ArrowRight className="w-4 h-4" />
                                            </span>
                                        </button>
                                    </div>
                                </motion.div>
                            )}

                            {status === 'loading' && (
                                <motion.div
                                    key="loading"
                                    initial={{ opacity: 0, scale: 0.95 }}
                                    animate={{ opacity: 1, scale: 1 }}
                                    exit={{ opacity: 0 }}
                                    className="flex flex-col items-center justify-center py-12 text-slate-900 relative z-10"
                                >
                                    <CircleNotch className="w-10 h-10 animate-spin text-slate-900 mb-6" weight="bold" />
                                    <h3 className="text-xl font-bold tracking-tight">Securing Session</h3>
                                    <p className="text-sm text-slate-500 mt-2 text-center">Negotiating transaction protocol with server...</p>
                                </motion.div>
                            )}

                            {status === 'pending' && ticketId && (
                                <motion.div
                                    key="pending"
                                    initial={{ opacity: 0, scale: 0.95 }}
                                    animate={{ opacity: 1, scale: 1 }}
                                    className="relative z-10 flex flex-col items-center"
                                >
                                    <div className="w-full flex justify-between items-center mb-6">
                                        <div className="flex items-center gap-2">
                                            <CreditCard className="text-slate-400 w-5 h-5" />
                                            <span className="text-sm font-semibold text-slate-900">₹{amount}</span>
                                        </div>
                                        <div className="text-xs font-mono bg-slate-100 text-slate-500 px-2 py-1 rounded-md">
                                            ID: {ticketId.slice(0, 8)}...
                                        </div>
                                    </div>

                                    <div
                                        id="qr-code-container"
                                        className="bg-white p-2 rounded-3xl shadow-[0_0_0_1px_rgba(0,0,0,0.05),0_8px_16px_-4px_rgba(0,0,0,0.1)] mb-8"
                                    >
                                        <QRCodeSVG value={upiUrl} size={220} level="H" includeMargin={true} />
                                    </div>

                                    <div className="w-full space-y-3">
                                        <a
                                            href="intent://pay?#Intent;scheme=upi;end"
                                            className="flex items-center justify-center gap-2 w-full py-4 bg-slate-950 text-white font-semibold rounded-[1.25rem] shadow-md transition-all active:scale-[0.98] hover:bg-slate-900"
                                        >
                                            <Lightning fill="currentColor" className="w-5 h-5" />
                                            Pay via UPI App
                                        </a>

                                        <button
                                            onClick={downloadQR}
                                            className="flex items-center justify-center gap-2 w-full py-4 bg-white text-slate-700 font-semibold rounded-[1.25rem] border border-slate-200 transition-all active:scale-[0.98] hover:bg-slate-50"
                                        >
                                            <DownloadSimple className="w-5 h-5" />
                                            Save QR Code
                                        </button>
                                    </div>

                                    <div className="mt-8 flex items-center gap-2.5 px-4 py-2 bg-slate-50 text-slate-600 rounded-full text-sm font-medium">
                                        <span className="relative flex h-2.5 w-2.5">
                                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-slate-400 opacity-75"></span>
                                            <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-slate-500"></span>
                                        </span>
                                        Awaiting Confirmation...
                                    </div>
                                </motion.div>
                            )}

                            {status === 'paid' && (
                                <motion.div
                                    key="paid"
                                    initial={{ opacity: 0, scale: 0.9 }}
                                    animate={{ opacity: 1, scale: 1 }}
                                    className="flex flex-col items-center justify-center py-10 text-emerald-600 relative z-10"
                                >
                                    <motion.div
                                        initial={{ scale: 0 }}
                                        animate={{ scale: 1 }}
                                        transition={{ type: "spring", stiffness: 200, damping: 15 }}
                                    >
                                        <CheckCircle weight="fill" className="w-20 h-20 mb-6" />
                                    </motion.div>
                                    <h3 className="text-2xl font-bold tracking-tight text-slate-900">Payment Verified</h3>
                                    <p className="text-sm text-slate-500 mt-2">Issuing your ticket...</p>
                                </motion.div>
                            )}
                        </AnimatePresence>
                    </motion.div>
                </div>
            </div>
        </div>
    );
};

export default RegisterEvent;
