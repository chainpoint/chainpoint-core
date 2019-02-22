const env = require('../parse-env.js')('api')
const _ = require('lodash')
const restify = require('restify')
const connections = require('../connections.js')

async function getCoresRandomAsync (req, res, next) {
    try {
        rpc = connections.openTendermintConnection(env.TENDERMINT_URI)
        netInfo = await rpc.netInfo({})
    }catch (error){
        console.log(error)
        console.error('rpc error')
        return next(new restify.InternalServerError('Could not get net info'))
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

async function getCoreStatusAsync (req, res, next) {
    try {
        rpc = connections.openTendermintConnection(env.TENDERMINT_URI)
        status = await rpc.status({})
    } catch (error) {
        console.log(error)
        console.error('rpc error')
        return next(new restify.InternalServerError('Could not query for status'))
    }
    if (!status) {
        res.status(404)
        res.noCache()
        res.send({ code: 'NotFoundError', message: '' })
        return next()
    }
    res.noCache()
    res.contentType = 'application/json'
    res.cache('public', { maxAge: 1000 })
    res.send(status)
    return next()
}

module.exports = {
    getCoresRandomAsync: getCoresRandomAsync,
    getCoreStatusAsync: getCoreStatusAsync
}