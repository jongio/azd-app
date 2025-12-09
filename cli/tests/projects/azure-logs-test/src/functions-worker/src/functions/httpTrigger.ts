import { app, HttpRequest, HttpResponseInit, InvocationContext } from "@azure/functions";

const SERVICE_NAME = process.env.SERVICE_NAME || 'functions-worker';

/**
 * HTTP trigger function - Health check endpoint
 */
app.http('health', {
    methods: ['GET'],
    authLevel: 'anonymous',
    route: 'health',
    handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponseInit> => {
        context.log(`[INFO] Health check invoked - ${SERVICE_NAME}`);
        
        return {
            status: 200,
            jsonBody: {
                status: 'healthy',
                service: SERVICE_NAME,
                functionName: 'health',
                timestamp: new Date().toISOString()
            }
        };
    }
});

/**
 * HTTP trigger function - Generate sample logs
 */
app.http('generateLogs', {
    methods: ['GET', 'POST'],
    authLevel: 'anonymous',
    route: 'generate-logs',
    handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponseInit> => {
        const countParam = request.query.get('count');
        const count = countParam ? parseInt(countParam) : 5;
        
        const levels = ['INFO', 'WARN', 'ERROR', 'DEBUG'];
        
        for (let i = 0; i < count; i++) {
            const level = levels[Math.floor(Math.random() * levels.length)];
            const message = `Sample log message ${i + 1} of ${count} from ${SERVICE_NAME}`;
            
            switch (level) {
                case 'INFO':
                    context.log(message);
                    break;
                case 'WARN':
                    context.warn(message);
                    break;
                case 'ERROR':
                    context.error(message);
                    break;
                default:
                    context.trace(message);
            }
        }
        
        return {
            status: 200,
            jsonBody: { generated: count, service: SERVICE_NAME, functionName: 'generateLogs' }
        };
    }
});

/**
 * HTTP trigger function - Simulate error
 */
app.http('error', {
    methods: ['GET'],
    authLevel: 'anonymous',
    route: 'error',
    handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponseInit> => {
        context.error(`Simulated error in ${SERVICE_NAME} - this is a test error for log streaming`);
        
        return {
            status: 500,
            jsonBody: { error: 'Simulated error', service: SERVICE_NAME, functionName: 'error' }
        };
    }
});

/**
 * Timer trigger function - Periodic log generation for testing
 * Runs every 5 minutes
 */
app.timer('periodicLogger', {
    schedule: '0 */5 * * * *',
    handler: async (myTimer: any, context: InvocationContext): Promise<void> => {
        context.log(`[INFO] Periodic logger invoked at ${new Date().toISOString()}`);
        context.log(`[INFO] Service: ${SERVICE_NAME}, Timer scheduled next: ${myTimer.scheduleStatus?.next}`);
        
        // Generate some sample logs
        for (let i = 0; i < 3; i++) {
            context.log(`[INFO] Periodic log entry ${i + 1}/3 from ${SERVICE_NAME}`);
        }
    }
});

/**
 * HTTP trigger function - Root endpoint
 */
app.http('root', {
    methods: ['GET'],
    authLevel: 'anonymous',
    route: '',
    handler: async (request: HttpRequest, context: InvocationContext): Promise<HttpResponseInit> => {
        context.log(`[INFO] Root endpoint accessed - ${SERVICE_NAME}`);
        
        return {
            status: 200,
            jsonBody: {
                service: SERVICE_NAME,
                host: 'function',
                message: 'Azure Functions log streaming test service',
                timestamp: new Date().toISOString(),
                endpoints: [
                    'GET /api/health - Health check',
                    'GET /api/generate-logs?count=N - Generate N log entries',
                    'GET /api/error - Simulate error'
                ]
            }
        };
    }
});
