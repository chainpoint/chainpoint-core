const connections = require('./lib/connections.js')

async function getCoresRandomAsync (req, res, next) {
    try {
        rpc = connections.openTendermintConnection()
        netInfo = await rpc.netInfo({})
    }catch (error){
        console.error('rpc error')
        return next(new restify.InternalServerError('Could not query for tx by hash'))
    }
    if (!netInfo) {
        res.status(404)
        res.noCache()
        res.send({ code: 'NotFoundError', message: '' })
        return next()
    }
    if (netInfo.peers.length > 0) {
        let decodedPeers = netInfo.peers.map((peer, index, arr) => {
            let byteArray = Array.prototype.slice.call(Buffer.from(peer.remote_ip, 'base64'), 0)
            let newBytes = byteArray.slice(-4)
            return newBytes[0].toString(10) + "." + newBytes[1].toString(10) + "." + newBytes[2].toString(10) + "." + newBytes[3].toString(10)
        })
        res.contentType = 'application/json'
        res.cache('public', { maxAge: 2592000 })
        res.send(decodedPeers)
        return next()
    }
    res.noCache()
    res.send([])
    return next()
}

module.exports = {
    getCoresRandomAsync: getCoresRandomAsync
}