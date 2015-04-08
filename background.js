function onBeforeRequestHandler(details) {
    // console.log("serving: ", details.url, details);
    switch (details.type) {
        case "main_frame":
            console.log("MAIN" , details.url);
            break;
        case "image":
            var m = matches_blacklist(blacklist, details.url);
            if (! m.accepted) {
                // console.log("block script " + details.url + ": " + m.reason);
                log_block(details, m.reason);
                return { "redirectUrl": chrome.extension.getURL("empty.png") };
            }
            log_allow(details, m.reason);
            break;
        case "sub_frame":
        case "stylesheet":
        case "script":
        case "xmlhttprequest":
            var m = matches_blacklist(blacklist, details.url);
            if (! m.accepted) {
                log_block(details, m.reason);
                // console.log("block script " + details.url + ": " + m.reason);
                // return { "redirectUrl": chrome.extension.getURL("empty.js") };
                return { "redirectUrl": "about://blank" };
            }
            log_allow(details, m.reason);
            break;
        case "object":
            log_allow(details, "");
            break;
        case "other":
            // extension, fonts.
            break;
    }
    return { "cancel": false };
}

chrome.webRequest.onBeforeRequest.addListener(
    onBeforeRequestHandler,
    {
        "urls": [
            "<all_urls>",
        ]
    },
    [ "blocking" ]
);

// matches_blacklist returns wether the url is listed in map bl, and the reason.
// Keys in bl are the domains, and values are either null, in which case the
// whole domain is blocked, or a list of path prefixes.
function matches_blacklist(bl, url) {
    var u = new URL(url);
    if (u.protocol === "chrome-extension:") {
        // extensions should be OK.
        return {accepted: true, reason: "extensions are always accepted"}
    }
    // exact match
    if (match_domain(bl, u.hostname, u.pathname)) {
        return {accepted: false, reason: "exact hostname in blacklist"}
    }
    var hosts = u.hostname.split(".");
    if (hosts[hosts.length-1] === "") {
        // hostname was "foo.bar.com."
        hosts.splice(-1, 1);
    }
    // drop the subdomain, we already checked for an exact match.
    hosts.splice(0, 1);
    for (; hosts.length > 0; hosts.splice(0, 1)) {
        var host = "." + hosts.join(".")
        if (match_domain(bl, host, u.pathname)) {
            return {accepted: false, reason: "hostname variant in blacklist"}
        }
    }
    return {accepted: true, reason: "no blacklist match"}
}

// match_domain is true is domain is a key in bl, and prefix matched.
function match_domain(bl, domain, prefix) {
    var prefixes = bl[domain];
    if (prefixes === undefined) {
        // not present
        return false;
    }
    if (prefixes === null) {
        // all prefixes are blocked
        return true;
    }
    // prefix match
    var i = 0;
    for (; i < prefixes.length; i++) {
        if (prefix.substr(0, prefixes[i].length) === prefixes[i]) {
            return true;
        }
    }
    return false;
}

function log_block(details, reason) {
    chrome.tabs.get(details.tabId, function(tab) {
        console.log("block " + details.type + " " + details.url + ": " + reason + ", src: " + tab.url);
    });
}

function log_allow(details, reason) {
    console.log("allow " + details.type + " " + details.url + ": " + reason);
}

blacklist = {};

function block(hostname, pages) {
    if (pages === undefined) {
        // block everything from this domain.
        blacklist[hostname] = null
    } else {
        blacklist[hostname] = pages;
    }
}

block("js-agent.newrelic.com");
block("edge.quantserve.com");
block("static.chartbeat.com");
block("stats.g.doubleclick.net");
block("www.googletagservices.com"); // no idea what this is
block("www.googletagmanager.com"); // no idea what this is
block("www.google-analytics.com");
block("www.google.com", ["/jsapi"]);
block("a.disquscdn.com", ["/count"]);
block(".disqus.com", ["/count"]); // http://thedissolve.disqus.com/count.js 
// block("collector-cdn.github.com"); // for _stats, mostly.
block(".gamer-network.net"); // src: RPS
block("b.scorecardresearch.com");
block(".taboola.com"); // src: RPS
block(".krxd.net"); // src: bbc.com
block("me-cdn.effectivemeasure.net"); // bbc.com
block(".ivwbox.de"); // bbc.com
block("pagead2.googlesyndication.com", ["/pagead/show_ads.js"])
block("apis.google.com", ["/js/plusone.js"])
block("connect.facebook.net")
block("graph.facebook.com")
block("www.googleadservices.com")
block(".captifymedia.com") // wired
block(".vdna-assets.com") // wired
block(".inspectlet.com") // wired
block("clc.stackoverflow.com")
block(".ioam.de") // spiegel.de
block(".meetrics.net") // spiegel.de
block(".adition.com") // spiegel.de
block("c.spiegel.de") // spiegel.de
block(".mxcdn.net") // spiegel.de
// TODO: kill image http://sa.bbc.co.uk/bbc/bbc/s
