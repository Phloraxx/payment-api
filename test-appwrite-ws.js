const WebSocket = require("ws");

fetch("https://payment-api.nerdpixel.workers.dev/api/ticket", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ amount: 100 }),
})
    .then((res) => res.json())
    .then((data) => {
        console.log("Created Ticket:", data);

        // Subscribing to ALL documents in the collection to see if the event fires
        const AppwriteChannels = `databases.697522750025b8e28c32.collections.payment.documents`;
        const wsUrl = `wss://backend.mulearnscet.in/v1/realtime?project=6948e3640018a902b864&channels[]=${AppwriteChannels}`;

        console.log("Connecting to Appwrite Natively NATIVELY:", wsUrl);

        const ws = new WebSocket(wsUrl);

        ws.on("open", () => {
            console.log("✅ Appwrite WS Connected! Waiting 3 seconds to fire webhook...");

            setTimeout(() => {
                const payload = {
                    secret_key: "Chamaka123@@",
                    body: `TICKET${data.ticketId.replace("TICKET", "").trim()} SOURAV paid you ₹${data.amount}`,
                };
                console.log("Firing webhook:", payload);
                fetch("https://payment-api.nerdpixel.workers.dev/api/webhook", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(payload),
                })
                    .then((r) => r.json())
                    .then((r) => console.log("Webhook Response:", r));
            }, 3000);
        });

        ws.on("message", (msg) => {
            const p = JSON.parse(msg.toString());
            if (p.type === "event") {
                console.log("\n⚡ APPWRITE BROADCASTED TO THESE CHANNELS ⚡:\n", JSON.stringify(p.data.channels, null, 2));
                ws.close();
                process.exit(0);
            }
        });

        ws.on("error", (err) => console.error("WS Error:", err));
        ws.on("close", (code, reason) => console.log("WS Closed:", code, reason.toString()));
    })
    .catch(console.error);
