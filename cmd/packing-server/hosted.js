(async () => {
    const status = document.getElementById("access-status");
    const trip = location.pathname.split("/")[2];
    try {
        const access = new URLSearchParams(location.hash.slice(1)).get("access");
        if (access) {
            const response = await fetch("/api/v1/exchange", {
                method: "POST", credentials: "same-origin", cache: "no-store",
                headers: {"Content-Type": "application/json", "X-Packing-Request": "1"},
                body: JSON.stringify({secret: access}),
            });
            if (!response.ok) throw new Error("Access link was refused or has expired. Ask the operator for a fresh link.");
            history.replaceState(null, "", location.pathname);
        }
        const response = await fetch("/api/v1/access", {cache: "no-store", credentials: "same-origin"});
        if (!response.ok) throw new Error("Open the private access link to continue. The cleaned trip URL alone cannot authorize a new browser.");
        const scope = await response.json();
        if (scope.trip !== trip) throw new Error("This access link does not authorize this trip.");
        // Origin and tab isolation come from sessionStorage; grant scope isolates rotated access.
        // Epoch is retained in the envelope so a restore can offer recovery.
        const key = `packing:hosted:v1:${scope.scope}:${trip}`;
        let recovery = null, storageError = null;
        try { recovery = sessionStorage.getItem(key); } catch (_) { storageError = "Device recovery storage is unavailable."; }
        const client = Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, "0")).join("");
        const app = Elm.HostedMain.init({node: document.getElementById("app"), flags: {trip, client, now: Date.now(), recovery, storageError}});
        app.ports.storeRecovery.subscribe(state => {
            let error = null;
            try { sessionStorage.setItem(key, state); } catch (_) { error = "Could not save recovery on this device. Keep this tab open or export device recovery."; }
            app.ports.recoveryResult.send({state, error});
        });
        app.ports.exportRecovery.subscribe(state => {
            const blob = new Blob([state], {type: "application/json"});
            const url = URL.createObjectURL(blob), link = document.createElement("a");
            link.href = url; link.download = "packing-device-recovery.json"; link.click();
            setTimeout(() => URL.revokeObjectURL(url), 1000);
        });
        const refresh = () => app.ports.refreshRequested.send(null);
        document.addEventListener("visibilitychange", () => { if (document.visibilityState === "visible") refresh(); });
        window.addEventListener("pageshow", refresh);
        window.addEventListener("online", refresh);
        status.remove();
    } catch (error) {
        status.textContent = error instanceof Error ? error.message : "Could not open this private trip.";
    }
})();
