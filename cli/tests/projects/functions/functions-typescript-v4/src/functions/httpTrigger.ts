import { app, HttpRequest, HttpResponseInit, InvocationContext } from '@azure/functions';

interface HelloResponse {
    message: string;
    timestamp: string;
    method: string;
    name: string;
}

export async function httpTrigger(request: HttpRequest, context: InvocationContext): Promise<HttpResponseInit> {
    context.log('TypeScript HTTP trigger function processed a request.');
    
    const name = request.query.get('name') || await request.text() || 'World';
    
    const response: HelloResponse = {
        message: `Hello, ${name}!`,
        timestamp: new Date().toISOString(),
        method: request.method,
        name: name
    };
    
    return {
        status: 200,
        jsonBody: response
    };
}

app.http('httpTrigger', {
    methods: ['GET', 'POST'],
    authLevel: 'anonymous',
    handler: httpTrigger
});
