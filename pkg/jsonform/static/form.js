(function () {
    "use strict";

    /**
     * Form
     * @constructor
     */
    function JSONForm() {
        this.form = null;
        this.title = null;

        this.schemaUrl = '';
        this.schema = undefined;

        this.valueUrl = '';
        this.value = undefined;

        this.submitUrl = '';
        this.submitMethod = 'POST';

        this.fail = function (html) {
            alert(html)
        }

        this.success = function (html) {
            alert(html)
        }
    }

    /**
     * @typedef formQueryParams
     * @type {Object}
     * @property {String} title - Title of the form.
     * @property {String} schemaUrl - URL to fetch schema.
     * @property {String} valueUrl - URL to fetch value.
     * @property {String} submitUrl - URL to submit form.
     * @property {String} submitMethod - HTTP method to use on form submit.
     */


    JSONForm.prototype.default = function () {
        /**
         * @type {formQueryParams}
         */
        var params = this.queryParams()

        this.title = $("#title");

        if (params.title) {
            $(this.title).text(params.title)
            document.title = params.title
        }
        console.log("QUERY PARAMS:", params)

        this.fail = function (html) {
            $('#res').html('ERROR: ' + html)
        }

        this.success = function (html) {
            $('#res').html(html)
        }

        if (params.schemaUrl === null) {
            this.fail("Missing schemaUrl parameter in URL")
            return
        }

        if (params.submitUrl === null) {
            this.fail("Missing submitUrl parameter in URL")
            return
        }

        if (params.submitMethod !== null) {
            this.submitMethod = params.submitMethod;
        }

        this.submitUrl = params.submitUrl
        this.schemaUrl = params.schemaUrl

        if (typeof params.valueUrl === undefined) {
            this.valueUrl = params.valueUrl;
        }

        this.setFormElement($('#schema-form'))

        console.log("FORM", this)

        this.render()
    }

    JSONForm.prototype.render = function () {
        if (this.form === null) {
            this.fail("Missing destination form element, did you call setFormElement?")
            return
        }

        var self = this

        if (this.schema === undefined) {
            send(this.schemaUrl, "GET", null, 200, function (schema) {
                self.schema = schema;

                if (self.title !== null && $(self.title).text() === '') {
                    $(self.title).text(schema.schema.title)
                    document.title = schema.schema.title
                }

                self.render()
            }, function (x) {
                self.fail("Failed to load schema using URL:<br /><code>" + self.schemaUrl + "</code><br />Response:<br /><code>" + x.responseText + "</code>")
            })

            return
        }

        if (this.value === undefined && this.valueUrl !== null && this.valueUrl !== '') {
            send(this.valueUrl, "GET", null, 200, function (schema) {
                self.value = value;

                self.render()
            }, function (x) {
                self.fail("Failed to load value using URL:<br /><code>" + self.valueUrl + "</code><br />Response:<br /><code>" + x.responseText + "</code>")
            })

            return
        }


        console.log("Rendering form")

        var formConf = {
            schema: this.schema.schema,
            form: this.schema.form,
            onSubmit: function (errors, values) {
                this.success('')

                console.log("VALUES", values);
                console.log("ERRORS", errors);

                if (errors) {
                    console.log(errors)
                    return false;
                }

                if (self.submitUrl && self.submitMethod) {
                    send(self.submitUrl, self.submitMethod, values, 0, function () {
                        self.success('Submitted.')
                    }, function (x) {
                        self.fail("Failed to submit form using URL:<br /><code>" + self.submitUrl + "</code><br />Response:<br /><code>" + x.responseText + "</code>")
                    })
                }

                return false;
            }
        }

        if (typeof this.value !== undefined) {
            formConf.value = this.value
        }

        $(this.form).jsonForm(formConf);
    }

    /**
     * @param {Element} title - Form HTML element.
     */
    JSONForm.prototype.setTitleElement = function (title) {
        this.title = title;
    }


    /**
     * @param {Element} form - Form HTML element.
     */
    JSONForm.prototype.setFormElement = function (form) {
        this.form = form;
    }

    /**
     * Get URL query params as a map.
     * @return {Object}
     */
    JSONForm.prototype.queryParams = function () {
        var query = window.location.search.substring(1);

        return query.split('&').reduce(function (res, item) {
            var parts = item.split('=');
            res[parts[0]] = decodeURIComponent(parts[1]);
            return res;
        }, {})
    }

    /**
     * Set schema source with a URL.
     * @param {string} url
     */
    JSONForm.prototype.setSchemaUrl = function (url) {
        this.schemaUrl = url;
    }

    /**
     * Set schema source with a URL.
     * @param {string} url
     */
    JSONForm.prototype.setValueUrl = function (url) {
        this.valueUrl = url;
    }

    JSONForm.prototype.setSubmitURL = function (method, url) {
        this.submitMethod = method;
        this.submitUrl = url;
    }


    /**
     * @callback RawCallback
     * @param {XMLHttpRequest} value
     */

    /**
     * @callback JSONCallback
     * @param {Object} value
     */

    /**
     *
     * @param {String} url
     * @param {String} method
     * @param {Object} bodyValues
     * @param {Number} successStatus
     * @param {JSONCallback} successCallback
     * @param {RawCallback} failCallback
     */
    function send(url, method, bodyValues, successStatus, successCallback, failCallback) {
        var x = new XMLHttpRequest();
        x.onreadystatechange = function () {
            if (x.readyState !== XMLHttpRequest.DONE) {
                return;
            }

            if (!successStatus) {
                if (typeof (successCallback) === 'function') {
                    successCallback(x);
                }

                return
            }

            switch (x.status) {
                case successStatus:
                    if (typeof (successCallback) === 'function') {
                        successCallback(JSON.parse(x.responseText));
                    }
                    break;
                default:
                    if (typeof (failCallback) === 'function') {
                        failCallback(x)
                    } else {
                        throw {err: 'unexpected response', data: x};
                    }
            }
        };


        x.open(method, url, true);
        if (typeof bodyValues !== null) {
            x.setRequestHeader("Content-Type", "application/json; charset=utf-8");
            x.send(JSON.stringify(bodyValues));
            return;
        }

        x.send();
    }

    window.JSONForm = JSONForm;
})();