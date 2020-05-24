const http = require('http')
const express = require('express')
const uuid = require('uuid')

const app = express()
app.use(express.json())

const store = new Map() // FIXME: persistent store
store.set('test', 'test') // FIXME: remove test credentials

// returns the manifestID for the given stream key. 401s if no such event exists.
app.post('/auth', (req, res) => {
    if (!req.body || !req.body.StreamKey || !store.has(req.body.StreamKey)) {
        res.statusCode = 401
        return res.end()
    }
    const manifestID = store.get(req.body.StreamKey)
    res.statusCode = 200
    res.write(JSON.stringify({ ManifestID: manifestID }))
    res.end()
})

// creates a new manifestID+streamKey pair
app.get('/new', (_, res) => {
    const manifestID = uuid.v4()
    const streamKey = uuid.v4()
    store.set(streamKey, manifestID)
    res.statusCode = 200
    res.write(JSON.stringify({ streamKey, manifestID }))
    res.end()
})

app.server = http.createServer(app)
app.server.listen(8001)

console.log('Auth server running...')