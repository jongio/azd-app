const { app } = require('@azure/functions');

app.http('http', {
    methods: ['GET'],
    authLevel: 'anonymous',
    handler: async (req) => ({ status: 200, body: 'Valid function but corrupt host.json' })
});
