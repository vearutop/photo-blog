(function () {
    var visitorCookie = Cookies.get("v")

    function randomInt() {
        return Math.floor(Math.random()*(9223372036854775807-1+1)+1);
    }

    /**
     *
     * @type {{
     *   id: string,
     *   lowRes: boolean,
     * }}
     */
    var visitorData= JSON.parse(localStorage.getItem("visitorData"))

    if (visitorData && visitorData.id) {
        if (visitorCookie !== visitorData.id) {
            Cookies.set("v", visitorData.id, {expires:3*365, sameSite: "Strict"})
        }
    } else {
        visitorData = {}

        if (visitorCookie) {
            visitorData.id = visitorCookie
        } else {
            visitorData.id = randomInt().toString(36)
        }

        Cookies.set("v", visitorData.id, {expires:3*365, sameSite: "Strict"})
        localStorage.setItem("visitorData", JSON.stringify(visitorData))
    }

    window.visitorData = visitorData;
})()

/**
 * @param v {Boolean}
 */
function setLowRes(v) {
    window.visitorData.lowRes = v
    localStorage.setItem("visitorData", JSON.stringify(window.visitorData))
}

/**
 * Format bytes as human-readable text.
 *
 * @param bytes Number of bytes.
 * @param si True to use metric (SI) units, aka powers of 1000. False to use
 *           binary (IEC), aka powers of 1024.
 * @param dp Number of decimal places to display.
 *
 * @return Formatted string.
 */
function humanFileSize(bytes, si = false, dp = 1) {
    const thresh = si ? 1000 : 1024;

    if (Math.abs(bytes) < thresh) {
        return bytes + ' B';
    }

    const units = si
        ? ['kB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
        : ['KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB'];
    let u = -1;
    const r = 10 ** dp;

    do {
        bytes /= thresh;
        ++u;
    } while (Math.round(Math.abs(bytes) * r) / r >= thresh && u < units.length - 1);


    return bytes.toFixed(dp) + ' ' + units[u];
}