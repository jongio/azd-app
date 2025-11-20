module.exports = async function (context, req) {
    context.log('JavaScript HTTP trigger function processed a request (v3 model).');

    const name = (req.query.name || (req.body && req.body.name)) || 'World';
    
    context.res = {
        status: 200,
        body: {
            message: `Hello, ${name}! (v3 legacy model)`,
            timestamp: new Date().toISOString(),
            method: req.method
        },
        headers: {
            'Content-Type': 'application/json'
        }
    };
};
