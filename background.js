var log_daemon = "http://localhost:1709";
var blacklistReload = 1 * 60 * 1000;

// tabID -> last loaded URL.
var activeURL = {};

function onBeforeRequestHandler(details) {
    // console.log("serving: ", details.url, details);
    if (has_prefix(details.url, log_daemon)) {
        return;
    }

    switch (details.type) {
        case "main_frame":
            // console.log("MAIN" , details.url);
            activeURL[details.tabId] = details.url;
            break;
        case "image":
            var m = matches_blacklist(blacklist, details.url);
            if (! m.accepted) {
                // console.log("block script " + details.url + ": " + m.reason);
                log("block", details, m.reason);
                return { "redirectUrl": chrome.extension.getURL("empty.png") };
            }
            log("allow", details, m.reason, activeURL[details.tabId]);
            break;
        case "sub_frame":
        case "stylesheet":
        case "script":
        case "xmlhttprequest":
            // There are rumors syncronous xmlhttprequest are not handled here.

            var m = matches_blacklist(blacklist, details.url);
            if (! m.accepted) {
                log("block", details, m.reason, activeURL[details.tabId]);
                // console.log("block script " + details.url + ": " + m.reason);
                // return { "redirectUrl": chrome.extension.getURL("empty.js") };
                return { "redirectUrl": "about://blank" };
            }
            log("allow", details, m.reason, activeURL[details.tabId]);
            break;
        case "object":
            log("allow", details, "an object", activeURL[details.tabId]);
            break;
        case "other":
            log("allow", details, "other", activeURL[details.tabId]);
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
    var paths = bl[domain];
    if (paths === undefined) {
        // domain not present
        return false;
    }
    if (paths === null) {
        // all paths are blocked
        return true;
    }
    // prefix match
    var i = 0;
    for (; i < prefixes.length; i++) {
        if (has_prefix(prefix, prefixes[i])) {
            return true;
        }
    }
    return false;
}

function has_prefix(s, prefix) {
    return s.substr(0, prefix.length) === prefix
}

function log(action, details, reason, tabURL) {
    var req = new XMLHttpRequest();
    req.open('POST', log_daemon + "/log");
    req.send(JSON.stringify({
        "action": action,
        "type": details.type,
        "url": details.url,
        "reason": reason,
        "tabId": details.tabId,
        "tab": tabURL,
    }));
    // some error checks here might be nice.
}


blacklist = {};

function load_blacklist() {
    new_bl = {}
    var req = new XMLHttpRequest();
    req.onreadystatechange = function () {
        if (this.readyState === 4){
            // todo: check status code
            entries = JSON.parse(this.response);
            for (var i = 0; i < entries.length; i++) {
                // console.log("block", entries[i]);
                new_bl[entries[i]] = null;
            }
            blacklist = new_bl;
        }
    };
    req.open('GET', log_daemon + "/list");
    req.send(null);
}

load_blacklist();
setInterval(load_blacklist, blacklistReload);

/*
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
block("b.scorecardresearch.com");
block(".google-analytics.com");
*/
