/*globals $:false, Backend:false, console:false, Viz:false, svgPanZoom:false */

(function () {
    "use strict";

    /**
     * @constructor
     */
    function App() {
        var d = getJsonFromUrl();
        console.log(d);

        this.aggregate = d.aggregate || '{}';
        this.graphLimit = d.graphLimit || 100;
        this.graphPriority = d.graphPriority || 'wt';
        this.rootSymbol = d.rootSymbol || 'main()';     // Graph root symbol.
        this.symbol = d.symbol || this.rootSymbol;      // Info pane symbol.
        this.showTop = d.showTop || 0;                  // Show top traces flag.
        this.tracesLimit = d.tracesLimit || 5;          // Limit top traces count.

        this.svgData = null;

        this.client = new Backend('');
        this.client.prepareRequest = function (request) {
            request.setRequestHeader("X-Foo", "bar");
        };

        var self = this;

        $('#symbol-name').text(this.rootSymbol);

        $('#aggregate input[name=aggregate]').val(this.aggregate);
        $('#aggregate input[name=rootSymbol]').val(this.rootSymbol);
        $('#aggregate input[name=tracesLimit]').val(this.tracesLimit);
        $('#aggregate input[name=graphLimit]').change(function () {
            self.graphLimit = $('#aggregate input[name=graphLimit]').val();
        }).val(this.graphLimit);
        $('#aggregate select[name=graphPriority]').change(function () {
            self.graphPriority = $('#aggregate select[name=graphPriority]').val();
        }).val(this.graphPriority);

        tpl.stat = $('script[data-template="symbolStatItem"]').text().split(/\${(.+?)}/g);
        tpl.trace = $('script[data-template="topTracesItem"]').text().split(/\${(.+?)}/g);
        tpl.recentItem = $('script[data-template="profileItem"]').text().split(/\${(.+?)}/g);
        tpl.aggregateItem = $('script[data-template="aggregateProfileItem"]').text().split(/\${(.+?)}/g);
    }

    var tpl = {};

    function render() {
        var props = arguments;
        return function (tok, i) {
            if (i % 2) {
                for (var a = 0; a < props.length; a++) {
                    if (typeof props[a][tok] !== "undefined") {
                        return props[a][tok];
                    }
                }
            }
            return tok;
        };
    }

    window.App = App;

    App.prototype.loadView = function () {
        if (this.showTop !== 0) {
            this.topTraces();
        } else {
            this.loadGraph();
        }

        this.loadSymbol();
    };

    App.prototype.updateURL = function () {
        var url = '?aggregate=' + encodeURIComponent(this.aggregate) + '&graphLimit=' + encodeURIComponent(this.graphLimit) + '&graphPriority=' + encodeURIComponent(this.graphPriority) + '&symbol=' + encodeURIComponent(this.symbol) + "&rootSymbol=" + encodeURIComponent(this.rootSymbol) + "&showTop=" + encodeURIComponent(this.showTop) + "&tracesLimit=" + encodeURIComponent(this.tracesLimit);

        window.history.pushState({}, document.title, url);
    };

    App.prototype.topTraces = function () {
        var self = this;

        this.tracesLimit = $('#aggregate input[name=tracesLimit]').val();

        this.client.getTopTraces({
            rootSymbol: self.rootSymbol,
            aggregate: JSON.parse(self.aggregate),
            limit: self.tracesLimit,
            resource: self.graphPriority
        }, function (traces) {
            console.log(traces);


            $("#graph").hide();

            var h = traces.map(function (item) {
                var val = {};
                val.traceStr = item.trace.join('<br>');

                var ii = {
                    symbol: item.symbol,
                    count: item.stat.ct,
                    aggregationSize: item.stat.as,
                    countFrac: item.stat.ctf,
                    wallTime: item.stat.wt,
                    wallTimeFrac: item.stat.wtf,
                    cpuTime: item.stat.cpu,
                    cpuTimeFrac: item.stat.cpuf,
                    ioTime: item.stat.io,
                    ioTimeFrac: item.stat.iof
                };

                val.statRow = tpl.stat.slice().map(render(ii)).join('');

                return tpl.trace.slice().map(render(item, val)).join('');
            });

            $("#topTraces").html('').append(h.join('')).show();
        }, function (err) {
            console.log(err.error);
        });

        this.updateURL();
    };

    App.prototype.saveSvg = function () {
        var preface = '<?xml version="1.0" standalone="no"?>\r\n';
        var svgBlob = new Blob([preface, this.svgData], {type: "image/svg+xml;charset=utf-8"});
        var svgUrl = URL.createObjectURL(svgBlob);
        var downloadLink = document.createElement("a");
        downloadLink.href = svgUrl;
        downloadLink.download = 'profile.svg'; // TODO elaborate name with current parameters.
        document.body.appendChild(downloadLink);
        downloadLink.click();
        document.body.removeChild(downloadLink);
    };

    App.prototype.focusGraph = function (symbol) {
        this.rootSymbol = symbol;
        $('#aggregate input[name=rootSymbol]').val(this.rootSymbol);
        $('#topTraces').hide();
        $('#graph').html('').show();
        this.loadGraph();
        this.updateURL();
    };

    App.prototype.setSymbol = function (symbol) {
        this.symbol = symbol;
        this.loadSymbol();
        this.updateURL();
    };

    App.prototype.loadSymbol = function () {
        var self = this;

        /**
         * @param {*} container
         * @param {Object<String, XhRenderValueStat>} values
         */
        function renderItems(container, values) {
            var items = [];

            for (var c in values) {
                if (!values.hasOwnProperty(c)) {
                    continue;
                }

                var item = values[c];

                items.push({
                    symbol: c,
                    count: item.ct,
                    aggregationSize: item.as,
                    countFrac: item.ctf,
                    wallTime: item.wt,
                    wallTimeFrac: item.wtf,
                    cpuTime: item.cpu,
                    cpuTimeFrac: item.cpuf,
                    ioTime: item.io,
                    ioTimeFrac: item.iof
                });
            }

            items = items.sort(function (a, b) {
                // console.log(a, b)
                if (parseFloat(a.wallTimeFrac) < parseFloat(b.wallTimeFrac)) {
                    return 1;
                }
                if (parseFloat(a.wallTimeFrac) > parseFloat(b.wallTimeFrac)) {
                    return -1;
                }

                return 0;
            });

            // console.log("sorted", items)

            container.html('').append(items.map(function (item) {
                return tpl.stat.slice().map(render(item)).join('');
            }).join(''));
        }

        this.client.getProfileSymbol({
            aggregate: JSON.parse(self.aggregate), symbol: self.symbol
        }, function (data) {
            // console.log(data)
            $('#symbol-name').text(self.symbol);
            renderItems($('#symbol-stat'), {inclusive: data.inclusive, exclusive: data.exclusive});
            // renderItems($('#symbol-stat'), {"inclusive": data.inclusive, "exclusive": data.exclusive})
            if (data.callees) {
                renderItems($('#callees'), data.callees);
                $('#callees-cont').show();
            } else {
                $('#callees-cont').hide();
            }

            if (data.callers) {
                renderItems($('#callers'), data.callers);
                $('#callers-cont').show();
            } else {
                $('#callers-cont').hide();
            }
        }, function (err) {
            $('#symbol-name').text("Aggregate not found.");
            console.log(err.error);
        });
    };

    App.prototype.loadRecentProfiles = function () {
        this.client.listProfiles({}, function (profiles) {
            $('#profiles').html('').append(profiles.recent.map(function (item) {
                var info = {};
                info.aggregate = JSON.stringify(item.addr);
                info.aggregateParam = encodeURIComponent(info.aggregate);
                info.id = item.addr.id;
                info.received =  new Date(item.addr.start * 1000).toISOString();

                return tpl.recentItem.slice().map(render(item, info)).join('');
            }).join(''));

            $('#activeAggregates').html('').append(profiles.activeAggregates.map(function (item) {
                var info = {}, d;
                info.aggregate = '';
                d = new Date(item.addr.start * 1000);
                info.aggregate += d.toISOString().substring(0, 13);
                d = new Date(item.addr.end * 1000);
                info.aggregate += " - " + d.toISOString().substring(0, 13);
                if (item.addr.labels) {
                    for (var l in item.addr.labels) {
                        info.aggregate += ", " + l + ": " + item.addr.labels[l];
                    }
                }
                // info.aggregate = JSON.stringify(item.addr);
                info.aggregateParam = encodeURIComponent(JSON.stringify(item.addr));
                return tpl.aggregateItem.slice().map(render(item, info)).join('');
            }).join(''));
        });
    };

    App.prototype.loadGraph = function () {
        var self = this;
        var graph = $('#graph');
        graph.html('<h2 style="text-align: center">loading graph...</h2>');

        $('#focus-graph').prop('disabled', true);
        $('#reset-graph').prop('disabled', true);

        function renderDot(code) {
            graph.html('<h2 style="text-align: center">rendering svg...</h2>');
            var viz = new Viz();

            viz.renderSVGElement(code)
                .then(function (element) {
                    element.setAttribute("xmlns", "http://www.w3.org/2000/svg");
                    self.svgData = element.outerHTML;

                    $('#graph').html('').append(element);

                    element.addEventListener('click', function (e) {
                        var symbol = $(e.target).closest('g.symbol').find('title').text();

                        if (symbol === '') {
                            return;
                        }

                        self.setSymbol(symbol);
                    });

                    svgPanZoom('#graph > svg', {
                        zoomEnabled: true, controlIconsEnabled: true, fit: true, center: true, minZoom: 0.1
                    });

                    $('#focus-graph').prop('disabled', false);
                    $('#reset-graph').prop('disabled', false);
                });
        }

        this.client.getProfileDot({
            rootSymbol: self.rootSymbol,
            graphLimit: self.graphLimit,
            graphPriority: self.graphPriority,
            aggregate: JSON.parse(self.aggregate)
        }, function (x) {
            renderDot(x.responseText);
        }, function (err) {
            $('#graph').html('<h2 style="text-align: center">not found</h2>');
            console.log(err);
        });

        this.updateURL();
    };

    function getJsonFromUrl(url) {
        if (!url) {
            url = location.href;
        }
        var question = url.indexOf("?");
        var hash = url.indexOf("#");
        if (hash === -1 && question === -1) {
            return {};
        }
        if (hash === -1) {
            hash = url.length;
        }
        var query = question === -1 || hash === question + 1 ? url.substring(hash) : url.substring(question + 1, hash);
        var result = {};
        query.split("&").forEach(function (part) {
            if (!part) {
                return;
            }
            part = part.split("+").join(" "); // replace every + with space, regexp-free version
            var eq = part.indexOf("=");
            var key = eq > -1 ? part.substr(0, eq) : part;
            var val = eq > -1 ? decodeURIComponent(part.substr(eq + 1)) : "";
            var from = key.indexOf("[");
            if (from === -1) {
                result[decodeURIComponent(key)] = val;
            } else {
                var to = key.indexOf("]", from);
                var index = decodeURIComponent(key.substring(from + 1, to));
                key = decodeURIComponent(key.substring(0, from));
                if (!result[key]) {
                    result[key] = [];
                }
                if (!index) {
                    result[key].push(val);
                } else {
                    result[key][index] = val;
                }
            }
        });
        return result;
    }

})();

