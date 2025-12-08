// Config service - no tests
module.exports = {
  apiUrl: process.env.API_URL || 'http://localhost:3000',
  debug: process.env.DEBUG === 'true',
}
