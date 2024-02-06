(function () {
    var visitorCookie = Cookies.get("v")
    console.log("VISITOR COOKIE", visitorCookie)

    function randomInt() {
        return Math.floor(Math.random()*(9223372036854775807-1+1)+1);
    }

    /**
     *
     * @type {{
     *   id: string,
     * }}
     */
    var visitorData= JSON.parse(localStorage.getItem("visitorData"))

    if (visitorData && visitorData.id) {
        if (visitorCookie !== visitorData.id) {
            console.log("SETTING VISITOR ID", visitorCookie, "TO", visitorData.id)
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

    console.log("VISITOR ID", visitorData.id)
})()