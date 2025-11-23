const { app } = require('@azure/functions');

app.http('http', {
    methods: ['GET'],
    authLevel: 'anonymous',
    handler: async (req) => ({ status: 200, body: 'This should not work without host.json' })
});
