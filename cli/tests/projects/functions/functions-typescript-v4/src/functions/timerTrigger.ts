import { app, Timer, InvocationContext } from '@azure/functions';

export function timerTrigger(myTimer: Timer, context: InvocationContext): void {
    const timeStamp: string = new Date().toISOString();
    
    context.log('TypeScript timer trigger function executed at:', timeStamp);
    
    if (myTimer.isPastDue) {
        context.log('Timer is running late!');
    }
    
    context.log('Next scheduled run:', myTimer.scheduleStatus?.next);
}

app.timer('timerTrigger', {
    schedule: '0 */5 * * * *',
    handler: timerTrigger
});
