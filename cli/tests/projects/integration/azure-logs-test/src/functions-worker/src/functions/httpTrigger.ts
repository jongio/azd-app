import { app, HttpRequest, HttpResponseInit, InvocationContext } from "@azure/functions";

const SERVICE_NAME = process.env.SERVICE_NAME || "functions-worker";

interface AzureProvenance {
    provider: "azure";
    service: "functions";
    siteName: string;
    region: string;
    hostname: string;
    runtime: string;
    sku: string;
    instanceId: string;
}

/**
 * Returns true when likely running in an Azure Functions environment.
 *
 * Note: local runners for `host: function` may set WEBSITE_HOSTNAME but not WEBSITE_SITE_NAME,
 * so we treat either as sufficient for emitting Azure provenance fields.
 */
function isAzureEnvironment(): boolean {
    return Boolean(
        process.env.WEBSITE_SITE_NAME ||
        process.env.WEBSITE_HOSTNAME ||
        process.env.WEBSITE_INSTANCE_ID
    );
}

function normalizeRegion(regionRaw: string | undefined): string {
    const region = (regionRaw || "unknown").trim();
    return region.replaceAll(/\s+/g, "-").toLowerCase();
}

function buildAzureProvenance(): AzureProvenance | null {
    if (!isAzureEnvironment()) return null;

    const siteName = process.env.WEBSITE_SITE_NAME || SERVICE_NAME;
    const regionRaw = process.env.REGION_NAME || process.env.AZURE_REGION || "unknown";

    return {
        provider: "azure",
        service: "functions",
        siteName,
        region: normalizeRegion(regionRaw),
        hostname: process.env.WEBSITE_HOSTNAME || "unknown",
        runtime: process.env.FUNCTIONS_WORKER_RUNTIME || "unknown",
        sku: process.env.WEBSITE_SKU || "unknown",
        instanceId: process.env.WEBSITE_INSTANCE_ID || "unknown",
    };
}

function formatAzureProvenance(provenance: AzureProvenance | null): string {
    if (!provenance) return "";
    return (
        `azure_provider=${provenance.provider} ` +
        `azure_service=${provenance.service} ` +
        `azure_site=${provenance.siteName} ` +
        `azure_region=${provenance.region} ` +
        `azure_hostname=${provenance.hostname} ` +
        `azure_runtime=${provenance.runtime} ` +
        `azure_sku=${provenance.sku} ` +
        `azure_instance=${provenance.instanceId}`
    );
}

function logEffectiveEndpoint(request: HttpRequest, context: InvocationContext, route: string): void {
    const xfHost = request.headers.get("x-forwarded-host") ?? request.headers.get("x-original-host");
    const xfProto = request.headers.get("x-forwarded-proto");
    const host = xfHost ?? request.headers.get("host");
    const proto = xfProto ?? "https";

    const url = request.url;
    const urlObj = new URL(url);
    const effectiveUrl = host ? `${proto}://${host}${urlObj.pathname}${urlObj.search}` : url;

    const envBits = [
        process.env.WEBSITE_HOSTNAME ? `WEBSITE_HOSTNAME=${process.env.WEBSITE_HOSTNAME}` : undefined,
        process.env.FUNCTIONS_WORKER_RUNTIME ? `FUNCTIONS_WORKER_RUNTIME=${process.env.FUNCTIONS_WORKER_RUNTIME}` : undefined,
    ].filter(Boolean).join(" ");

    const azureFields = formatAzureProvenance(buildAzureProvenance());

    context.log(
        `[ENDPOINT] service=${SERVICE_NAME} method=${request.method} route=${route} ` +
            `endpoint=${effectiveUrl} xf_host=${xfHost} xf_proto=${xfProto} ${envBits}` +
            (azureFields ? ` ${azureFields}` : "")
    );
}

// HTTP trigger - Health check
app.http("health", {
    methods: ["GET"],
    authLevel: "anonymous",
    route: "health",
    handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponseInit> => {
        logEffectiveEndpoint(request, context, "/api/health");
        context.log(`[INFO] Health endpoint hit - ${SERVICE_NAME} is healthy`);
        return {
            status: 200,
            jsonBody: {
                status: "healthy",
                service: SERVICE_NAME,
                functionName: "health",
                timestamp: new Date().toISOString(),
            },
        };
    },
});

// HTTP trigger - Generate logs
app.http("generateLogs", {
    methods: ["GET", "POST"],
    authLevel: "anonymous",
    route: "generate-logs",
    handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponseInit> => {
        logEffectiveEndpoint(request, context, "/api/generate-logs");
        const countParam = request.query.get("count");
        const count = countParam ? Number.parseInt(countParam, 10) : 5;
        const levels = ["INFO", "WARN", "ERROR", "DEBUG"] as const;

        for (let i = 0; i < count; i++) {
            const level = levels[Math.floor(Math.random() * levels.length)];
            const message = `Sample log message ${i + 1} of ${count} from ${SERVICE_NAME}`;
            switch (level) {
                case "INFO":
                    context.log(message);
                    break;
                case "WARN":
                    context.warn(message);
                    break;
                case "ERROR":
                    context.error(message);
                    break;
                default:
                    context.trace(message);
            }
        }

        return {
            status: 200,
            jsonBody: { generated: count, service: SERVICE_NAME, functionName: "generateLogs" },
        };
    },
});

// HTTP trigger - Simulate error
app.http("error", {
    methods: ["GET"],
    authLevel: "anonymous",
    route: "error",
    handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponseInit> => {
        logEffectiveEndpoint(request, context, "/api/error");
        context.error(`Simulated error in ${SERVICE_NAME} - this is a test error for log streaming`);
        return {
            status: 500,
            jsonBody: { error: "Simulated error", service: SERVICE_NAME, functionName: "error" },
        };
    },
});

// Timer trigger - Periodic log generation for testing
let timerCounter = 0;
app.timer("periodicLogger", {
    // NCRONTAB with seconds: fires every minute at second 0
    schedule: "0 * * * * *",
    handler: async (_myTimer: unknown, context: InvocationContext): Promise<void> => {
        timerCounter++;
        const azureFields = formatAzureProvenance(buildAzureProvenance());
        const azureSuffix = azureFields ? ` ${azureFields}` : "";

        // Emit a dedicated provenance line so dashboards/queries can reliably spot it.
        if (azureFields) {
            context.log(`[INFO] Azure Functions provenance:${azureSuffix}`);
        }

        context.log(`[INFO] Periodic logger invoked at ${new Date().toISOString()}${azureSuffix}`);
        context.log(`[INFO] Service: ${SERVICE_NAME}, iteration: ${timerCounter}${azureSuffix}`);

        const messages = [
            "Function processing scheduled task",
            "Background job completed successfully",
            "Queue message processed",
            "Timer trigger heartbeat - service healthy",
            "Scheduled maintenance check passed",
        ];
        const message = messages[Math.floor(Math.random() * messages.length)];
        context.log(`[INFO] ${message} - run #${timerCounter}${azureSuffix}`);

        if (timerCounter % 5 === 0) {
            context.warn(`[WARN] High latency detected at iteration ${timerCounter} - ${SERVICE_NAME}${azureSuffix}`);
        }
        if (timerCounter % 12 === 0) {
            context.error(
                `[ERROR] Transient storage timeout at iteration ${timerCounter} - ${SERVICE_NAME} (auto-retry succeeded)${azureSuffix}`
            );
        }
    },
});

// HTTP trigger - Root endpoint
app.http("root", {
    methods: ["GET"],
    authLevel: "anonymous",
    route: "",
    handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponseInit> => {
        logEffectiveEndpoint(request, context, "/api");
        const azureProvenance = buildAzureProvenance();
        const azureFields = formatAzureProvenance(azureProvenance);

        const baseUrl = azureProvenance
            ? `https://${azureProvenance.hostname}`
            : "http://localhost:7071";

        context.log(`[INFO] Root endpoint hit - Welcome to ${SERVICE_NAME}`);
        context.log(`[INFO] Public endpoint: GET ${baseUrl}/api (root)`);
        context.log(`[INFO] Public endpoint: GET ${baseUrl}/api/health (health check)`);
        context.log(`[INFO] Public endpoint: GET ${baseUrl}/api/generate-logs?count=N (generate logs)`);
        context.log(`[INFO] Public endpoint: GET ${baseUrl}/api/error (error simulation)`);

        if (azureProvenance) {
            context.log(`[INFO] Azure Functions provenance: ${azureFields}`);
        } else {
            context.log(`[INFO] Running locally (no Azure provenance)`);
        }

        return {
            status: 200,
            jsonBody: {
                service: SERVICE_NAME,
                host: "function",
                message: "Azure Functions log streaming test service",
                timestamp: new Date().toISOString(),
                endpoints: [
                    "GET /api/health - Health check",
                    "GET /api/generate-logs?count=N - Generate N log entries",
                    "GET /api/error - Simulate error",
                ],
                azure: azureProvenance
                    ? {
                        provider: azureProvenance.provider,
                        service: azureProvenance.service,
                        site: azureProvenance.siteName,
                        region: azureProvenance.region,
                        hostname: azureProvenance.hostname,
                    }
                    : null,
            },
        };
    },
});
