const http = require('http')
const express = require('express')

const app = express()
app.use(express.json())

app.post('/auth', (req, res) => {
    res.statusCode = 200
    res.write(JSON.stringify({ manifestID: "testManifestID" }))
    res.end()
})

app.server = http.createServer(app)
app.server.listen(8001)

console.log('Auth server running...')