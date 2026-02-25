import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import axios from 'axios';
import { motion } from 'framer-motion';
import { Ticket, CheckCircle, ArrowLeft, User, CurrencyInr, IdentificationCard } from '@phosphor-icons/react';

const API_URL = 'https://payment-api.nerdpixel.workers.dev/api';

interface TicketDetails {
    ticketId: string;
    status: string;
    amount: number;
    senderName: string;
}

const TicketSuccess = () => {
    const { id } = useParams();
    const [ticket, setTicket] = useState<TicketDetails | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        if (id) {
            axios.get(`${API_URL}/status/${id}`)
                .then(res => {
                    setTicket(res.data);
                    setLoading(false);
                })
                .catch(err => {
                    console.error(err);
                    setLoading(false);
                });
        }
    }, [id]);

    if (loading) {
        return (
            <div className="min-h-[100dvh] bg-[#f9fafb] flex items-center justify-center font-sans">
                <motion.div
                    initial={{ opacity: 0, scale: 0.9 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ repeat: Infinity, duration: 1, repeatType: "reverse" }}
                    className="w-16 h-16 bg-white rounded-2xl shadow-[0_20px_40px_-15px_rgba(0,0,0,0.05)] border border-slate-200/50 flex items-center justify-center"
                >
                    <Ticket className="w-8 h-8 text-slate-400" />
                </motion.div>
            </div>
        );
    }

    if (!ticket) {
        return (
            <div className="min-h-[100dvh] bg-[#f9fafb] flex items-center justify-center font-sans">
                <div className="text-slate-400 font-medium">Session expired or invalid transaction ID.</div>
            </div>
        );
    }

    return (
        <div className="min-h-[100dvh] bg-[#f9fafb] font-sans flex items-center justify-center p-6 md:p-12 overflow-hidden relative">

            {/* Ambient Base Layer */}
            <div className="absolute inset-0 bg-slate-100/30"></div>

            <div className="relative w-full max-w-md z-10 flex flex-col gap-6">

                {/* Primary Pass Surface */}
                <motion.div
                    initial={{ opacity: 0, y: 40 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ type: "spring", stiffness: 100, damping: 20 }}
                    className="bg-white rounded-[2.5rem] p-8 md:p-10 shadow-[0_30px_60px_-15px_rgba(0,0,0,0.08)] border border-slate-200/50 relative overflow-hidden group"
                >
                    {/* Integrated Top Highlight */}
                    <div className="absolute top-0 inset-x-0 h-1.5 bg-slate-900"></div>

                    <div className="flex flex-col items-center text-center">
                        <motion.div
                            initial={{ scale: 0 }}
                            animate={{ scale: 1 }}
                            transition={{ delay: 0.2, type: "spring", stiffness: 200, damping: 15 }}
                            className="w-16 h-16 bg-slate-50 text-slate-900 rounded-full flex items-center justify-center mb-6 ring-4 ring-white shadow-sm"
                        >
                            <CheckCircle weight="fill" className="w-10 h-10" />
                        </motion.div>

                        <h1 className="text-3xl font-bold tracking-tighter text-slate-950 mb-1">
                            Access Granted.
                        </h1>
                        <p className="text-sm font-medium text-slate-400 mb-10 tracking-wide uppercase">
                            Secure Verification Confirmed
                        </p>

                        <div className="w-full space-y-4">
                            {/* Bento Datapoints */}
                            <div className="flex items-center gap-4 bg-slate-50 p-4 rounded-2xl border border-slate-100 hover:border-slate-200 transition-colors">
                                <div className="w-10 h-10 rounded-full bg-white flex items-center justify-center shadow-sm">
                                    <User className="w-5 h-5 text-slate-900" />
                                </div>
                                <div className="text-left flex-1">
                                    <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400">Payer Identity</p>
                                    <p className="text-sm font-semibold text-slate-900">{ticket.senderName || 'Authorized Entity'}</p>
                                </div>
                            </div>

                            <div className="flex items-center gap-4 bg-slate-50 p-4 rounded-2xl border border-slate-100 hover:border-slate-200 transition-colors">
                                <div className="w-10 h-10 rounded-full bg-white flex items-center justify-center shadow-sm">
                                    <CurrencyInr className="w-5 h-5 text-slate-900" />
                                </div>
                                <div className="text-left flex-1">
                                    <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400">Total Cleared</p>
                                    <p className="text-sm font-semibold text-slate-900">₹{ticket.amount.toFixed(2)}</p>
                                </div>
                            </div>
                        </div>

                        <div className="w-full mt-10 pt-8 border-t border-dashed border-slate-200 relative">
                            {/* Cutout illusion on borders */}
                            <div className="absolute -left-11 -top-3 w-6 h-6 rounded-full bg-[#f9fafb] border-r border-slate-200"></div>
                            <div className="absolute -right-11 -top-3 w-6 h-6 rounded-full bg-[#f9fafb] border-l border-slate-200"></div>

                            <div className="flex flex-col items-center">
                                <IdentificationCard className="w-6 h-6 text-slate-300 mb-2" />
                                <p className="text-[10px] uppercase font-bold text-slate-400 tracking-widest mb-1">Pass Identifier</p>
                                <p className="text-sm font-mono text-slate-600 bg-slate-100 px-3 py-1.5 rounded-lg border border-slate-200">
                                    {ticket.ticketId}
                                </p>
                            </div>
                        </div>

                    </div>
                </motion.div>

                {/* Micro Actions */}
                <motion.div
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.4 }}
                    className="flex justify-center"
                >
                    <Link
                        to="/"
                        className="group flex items-center gap-2 text-sm font-medium text-slate-500 hover:text-slate-900 transition-colors bg-white px-5 py-3 rounded-full shadow-sm border border-slate-200/60"
                    >
                        <ArrowLeft className="w-4 h-4 group-hover:-translate-x-0.5 transition-transform" />
                        Initialize New Session
                    </Link>
                </motion.div>

            </div>
        </div>
    );
};

export default TicketSuccess;
