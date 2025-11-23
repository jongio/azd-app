const { app } = require('@azure/functions');

app.timer('timerTrigger', {
    schedule: '0 */5 * * * *',
    handler: (myTimer, context) => {
        const timeStamp = new Date().toISOString();
        
        context.log('Timer trigger function executed at:', timeStamp);
        
        if (myTimer.isPastDue) {
            context.log('Timer is running late!');
        }
        
        context.log(`Next scheduled run: ${myTimer.scheduleStatus.next}`);
    }
});
