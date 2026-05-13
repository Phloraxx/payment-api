const WebSocket = require("ws");

fetch("https://payment-api.nerdpixel.workers.dev/api/ticket", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ amount: 100 }),
})
    .then((res) => res.json())
    .then((data) => {
        console.log("Created Ticket:", data);

        const wsUrl = "wss://payment-api.nerdpixel.workers.dev/api/ws?ticketId=" + data.ticketId;
        console.log("Connecting to WebSocket:", wsUrl);
        const ws = new WebSocket(wsUrl);

        ws.on("open", () => {
            console.log("WebSocket Connected! Simulating payment in 8 seconds...");

            setTimeout(() => {
                const payload = {
                    secret_key: process.env.WEBHOOK_SECRET,
                    body: `TICKET${data.ticketId.replace("TICKET", "").trim()} SOURAV paid you ₹${data.amount}`,
                };
                console.log("Firing webhook payload:", payload);
                fetch("https://payment-api.nerdpixel.workers.dev/api/webhook", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(payload),
                })
                    .then((r) => r.json())
                    .then((r) => console.log("Webhook Response:", r));
            }, 8000);
        });

        ws.on("message", (msg) => {
            console.log("\n✅✅✅ WEBSOCKET PUSH RECEIVED FROM CLOUDFLARE ✅✅✅\n", msg.toString());
            ws.close();
            process.exit(0);
        });

        ws.on("close", (code, reason) => {
            console.log("WS Closed cleanly:", code, reason.toString());
            process.exit(0);
        });

        ws.on("error", (err) => {
            console.error("WS Error:", err);
            process.exit(1);
        });
    });
