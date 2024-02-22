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