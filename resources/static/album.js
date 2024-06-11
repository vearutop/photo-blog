/**
 * @param {XMLHttpRequest} x
 */
function onAlbumCreated(x) {
    /**
     * @type {PhotoAlbum}
     */
    var a = JSON.parse(x.responseText)
    window.location = "/edit/album/" + a.hash + ".html"
}

function deleteAlbum(name) {
    if (!window.confirm("Deleted album can not be restored")) {
        return
    }

    var b = new Backend('');
    b.controlDeleteAlbum({
        name: name,
    }, function (x) {
        alert("Album deleted, you'll be redirected to main page")
        window.location = "/"
    }, function (x) {
        alert("Failed to delete album: " + x.error)
    })
}

var featured = "featured"

function addToFeatured(imageHash) {
    var b = new Backend('');
    b.controlAddToAlbum({
        name: featured,
        body: {
            image_hash: imageHash
        }
    }, function () {
        // alert("Done")
    }, function (x) {
        alert("Failed: " + x.error)
    })
}

function removeImage(albumName, imageHash) {
    if (!window.confirm("This photo is about to be removed from the album '" + albumName + "'")) {
        return
    }

    var b = new Backend('');
    b.controlRemoveFromAlbum({
        name: albumName,
        hash: imageHash,
    }, function (x) {
        // alert("Photo is removed from the album")
    }, function (x) {
        alert("Failed to remove photo from the album: " + x.error)
    })
}

function toggleFullscreen() {
    if (document.fullscreenElement) {
        document.exitFullscreen()
        $('html body').css("overflow", "auto")
    } else {
        $('html body').css("overflow", "hidden")
        var el = $('html')[0]
        el.requestFullscreen()

        el.addEventListener("fullscreenchange", function () {
            if (!document.fullscreenElement) {
                $('html body').css("overflow", "auto")
            }
        });
    }
}

function exitFullscreen() {
    if (document.fullscreenElement) {
        document.exitFullscreen()
        $('html body').css("overflow", "auto")
    }
}

function enableDragNDropImagesReordering() {
    $("a.image").each(function () {
        var a = $(this)

        a.attr("draggable", true);
        a.on("dragstart", function (e) {
            console.log("dragstart", e)
            e.originalEvent.dataTransfer.setData("text/plain", a.data("hash"));
        })

        a.on("dragover", function (e) {
            e.preventDefault();
        })

        a.on("drop", function (e) {
            e.preventDefault();
            var draggedHash = e.originalEvent.dataTransfer.getData("text/plain");
            var afterTs = $(e.currentTarget).data('ts')
            console.log("dragdrop", "dragged hash", draggedHash, "after", afterTs, e);

            $(e.currentTarget).after($("#img" + draggedHash))
        });

    });

    $("div.chrono-text").each(function () {
        var div = $(this);

        div.on('drop', function (e) {
            e.preventDefault();
            var data = e.originalEvent.dataTransfer.getData("text/plain");
            console.log("dragdrop-text", data, e);

            $(e.currentTarget).after($("#img" + data))
        });

        div.on('dragover', function (e) {
            e.preventDefault();
        });

    });
}

(function () {
    if (screen.width > 576 || screen.width == 0) {
        return
    }

    var styles = `
@media screen and (orientation:portrait) {
    .main {
        margin-left: 0;
        margin-right: 0;
    }
    .thumb, a.image, .thumb canvas {
        width: ` + screen.width + `px;
        height: ` + Math.trunc(screen.width / 1.5) + `px;
    }
}
`

    var styleSheet = document.createElement("style")
    styleSheet.innerText = styles
    document.head.appendChild(styleSheet)
})()


/**
 * @typedef loadAlbumParams
 * @type {Object}
 * @property {String} albumName
 * @property {String} mapTiles
 * @property {String} mapAttribution
 * @property {Boolean} showMap
 * @property {UsecaseGetAlbumOutputCallback} albumData
 * @property {String} gallery - CSS selector for gallery container
 * @property {String} galleryPano - CSS selector for gallery panoramas container
 * @property {String} baseUrl - base address to set on image close
 * @property {String} imageBaseUrl - base address to link to full-res images
 */

function collectStats(params) {
    params.sw = screen.width
    params.sh = screen.height
    params.px = window.devicePixelRatio
    params.v = window.visitorData.id

    $.get("/stats", params)
}

function collectThumbVisibility() {
    var visibleSince = {}
    var visibleFor = {}
    var lastFlush = new Date()

    var options = {threshold: 1.0};
    var observer = new IntersectionObserver(function (entries) {
        var now = new Date();
        for (i in entries) {
            var e = entries[i];
            var h = $(e.target).data('hash')
            if (e.isIntersecting) {
                if (!visibleSince[h]) {
                    visibleSince[h] = now
                }
            } else {
                if (visibleSince[h]) {
                    if (!visibleFor[h]) {
                        visibleFor[h] = now - visibleSince[h];
                    } else {
                        visibleFor[h] += now - visibleSince[h];
                    }

                    delete (visibleSince[h])
                }
            }
        }

        if (now - lastFlush >= 5000) {
            collectStats({"thumb": JSON.stringify(visibleFor)})
            visibleFor = {}
            lastFlush = now
        }
    }, options);


    $("a.image").each(function (i, e) {
        observer.observe(e);
    })
}

/**
 *
 * @param {loadAlbumParams} params
 */
function loadAlbum(params) {
    "use strict";

    if (params.albumName === "") {
        return;
    }

    var gpsBounds = {
        minLat: null,
        maxLat: null,
        minLon: null,
        maxLon: null
    };

    if (params.mapTiles == "") {
        params.mapTiles = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
    }

    if (params.mapAttribution == "") {
        params.mapAttribution = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
    }

    if (params.gallery == "") {
        params.gallery = "#gallery"
    }

    if (params.gallery == "") {
        params.gallery = "#gallery-pano"
    }

    /**
     * @type {Array<PhotoImage>}
     */
    var gpsMarkers = []
    var client = new Backend('');

    var originalPath = params.baseUrl;
    if (originalPath == "") {
        originalPath = window.location.toString();
    }

    /**
     *
     * @param {UsecaseGetAlbumOutput} result
     */
    function renderAlbum(result) {
        console.log("RESULT", result.album.name, result.album.title, result)

        var hashByIdx = {}
        var idxByHash = {}
        var idx = 0
        var hideOriginal = result.hide_original

        if (typeof result.images === 'undefined') {
            result.images = [];
        }

        var albumSettings = result.album.settings
        var chronoTexts = result.album.settings.texts

        var fullscreenSupported = false
        if (typeof document.exitFullscreen == 'function') {
            fullscreenSupported = true
        }

        var exifHtml = {}

        for (var i = 0; i < result.images.length; i++) {
            var img = result.images[i]

            // Image is likely still being indexed.
            if (!img.blur_hash) {
                continue
            }

            if (typeof img.gps != 'undefined') {
                gpsMarkers.push(img)
                if (gpsBounds.maxLat == null) {
                    gpsBounds.maxLat = img.gps.latitude
                    gpsBounds.minLat = img.gps.latitude
                    gpsBounds.maxLon = img.gps.longitude
                    gpsBounds.minLon = img.gps.longitude
                } else {
                    if (img.gps.longitude > gpsBounds.maxLon) {
                        gpsBounds.maxLon = img.gps.longitude
                    }
                    if (img.gps.longitude < gpsBounds.minLon) {
                        gpsBounds.minLon = img.gps.longitude
                    }
                    if (img.gps.latitude > gpsBounds.maxLat) {
                        gpsBounds.maxLat = img.gps.latitude
                    }
                    if (img.gps.latitude < gpsBounds.minLat) {
                        gpsBounds.minLat = img.gps.latitude
                    }
                }
            }

            if (typeof img.is_360_pano == "undefined" || !img.is_360_pano) {
                hashByIdx[idx] = img.hash
                idxByHash[img.hash] = idx
                idx++

                var a = $("<a>")
                a.attr("id", 'img' + img.hash)
                a.attr("data-hash", img.hash)

                if (i < 4) {
                    a.attr("class", "image img" + i)
                } else {
                    a.attr("class", "image")
                }
                if (hideOriginal) {
                    a.attr("href", "#")
                } else {
                    if (params.imageBaseUrl) {
                        a.attr("href", params.imageBaseUrl + img.name)
                    } else {
                        a.attr("href", "/image/" + img.hash + ".jpg")
                    }
                }
                a.attr("target", "_blank")
                a.attr("data-idx", i)

                var srcSet = "/thumb/300w/" + img.hash + ".jpg 300w" +
                    ", /thumb/600w/" + img.hash + ".jpg 600w" +
                    ", /thumb/1200w/" + img.hash + ".jpg 1200w"


                if (!visitorData.lowRes) {
                    srcSet += ", /thumb/2400w/" + img.hash + ".jpg 2400w"
                }

                if (img.width > 0 && img.height > 0) {
                    if (!hideOriginal && !visitorData.lowRes) {
                        if (params.imageBaseUrl) {
                            srcSet += ", " + params.imageBaseUrl + img.name + " " + img.width + "w"
                        } else {
                            srcSet += ", /image/" + img.hash + ".jpg " + img.width + "w"
                        }
                    }

                    a.attr("data-pswp-width", img.width)
                    a.attr("data-pswp-height", img.height)
                }

                var img_description = '<a title="Edit details" class="control-panel ctrl-btn edit-icon" href="/edit/image/' + img.hash + '.html"></a>'
                if (result.album.name !== featured) {
                    img_description += '<a title="Add to featured" class="control-panel ctrl-btn star-icon" href="#" onclick="addToFeatured(\'' + img.hash + '\');return false"></a>'
                }
                img_description += '<a title="Remove from album" class="control-panel ctrl-btn trash-icon" href="#" onclick="removeImage(\'' + params.albumName + '\',\'' + img.hash + '\');return false"></a>'

                var exif = {}
                if (typeof img.exif !== "undefined") {
                    exif = img.exif
                }


                var ts = Date.parse(img.time)

                if (chronoTexts) {
                    var ct = [];

                    for (var ti = 0; ti < chronoTexts.length; ti++) {
                        var t = chronoTexts[ti]

                        var tt = Date.parse(t.time)

                        if (albumSettings.newest_first) {
                            if (tt < ts) {
                                ct.push(t)

                                continue
                            }
                        } else {
                            if (tt > ts) {
                                ct.push(t)

                                continue
                            }
                        }

                        var div = $("<div data-ts='" + t.time + "' class='chrono-text pure-g'><div class='text pure-u-3-5 some-text'>" + t.text + "</div></div>")

                        $(params.gallery).append(div)
                    }
                }

                chronoTexts = ct

                exif["file_name"] = img.name
                exif["size"] = humanFileSize(img.size) + ", " + (Math.round((img.width * img.height) / 10000) / 100 + " MP")

                if (typeof img.gps != 'undefined') {
                    exif["location"] =
                        '<a class="google-maps" href="https://maps.google.com/maps?q=loc:' +
                        img.gps.latitude.toFixed(8) + ',' + img.gps.longitude.toFixed(8) + '">google maps</a>' +
                        ' <a class="apple-maps" href="https://maps.apple.com/?ll=' +
                        img.gps.latitude.toFixed(8) + ',' + img.gps.longitude.toFixed(8) + '">apple maps</a>'
                }

                if (typeof img.meta !== "undefined") {
                    var cl = img.meta.image_classification
                    for (var ci in cl) {
                        var l = cl[ci]

                        exif[l.model + ":" + l.text] = l.score
                    }
                }

                var exh = '<table>'
                for (var k in exif) {
                    if (k === "hash" || k === "exposure_time_sec" || k === "created_at") {
                        continue;
                    }

                    var v = exif[k]

                    if (!v) {
                        continue;
                    }

                    exh += "<tr><th>" + k + "</th><td>" + v + "</td>";
                }
                exh += '</table>';

                exifHtml[img.hash] = exh

                if (img.description) {
                    img_description += '<div>' + img.description + '</div>';
                }

                if (typeof img.meta !== "undefined") {
                    img_description += '<div class="control-panel">'
                    var cl = img.meta.image_classification
                    for (var ci in cl) {
                        var l = cl[ci]

                        if (l.model === "cf-uform-gen2") {
                            continue
                        }

                        img_description += l.text + '<br/>'
                    }

                    if (img.meta.faces && img.meta.faces.length > 0) {
                        img_description += img.meta.faces.length + ' face(s)<br/>'
                    }

                    img_description += '</div>'

                    for (var ci in cl) {
                        var l = cl[ci]

                        if (l.model === "cf-uform-gen2") {
                            img_description += '<div style="margin-top: 20px"><span title="I don\'t always talk bullshit, but when I do, I\'m confident" class="icon-link robot-icon"></span><em>AI says:</em><br/>' + l.text + '</div>'
                        }
                    }

                    if (img.meta.geo_label) {
                        img_description += '<div style="margin-top: 20px"><span class="icon-link location-icon"></span>' + img.meta.geo_label + '</div>'
                    }
                }

                var landscape = ""
                if (img.width / img.height >= 1.499) {
                    landscape = " landscape"
                } else {
                    landscape = " portrait"
                }

                a.attr("data-pswp-srcset", srcSet)
                a.html('<div class="thumb' + landscape + '">' +
                    '<canvas id="bh-' + img.hash + '" width="32" height="32"></canvas>' +
                    '<img alt="photo" src="/thumb/200h/' + img.hash + '.jpg" srcset="/thumb/400h/' + img.hash + '.jpg 2x" /></div>')
                a.attr("data-ts", img.time)

                a.append('<div class="pswp-caption-content" style="display: none">' + img_description + '</div>')

                $(params.gallery).append(a)
                if (typeof img.blur_hash !== "undefined") {
                    blurHash(img.blur_hash, document.getElementById('bh-' + img.hash))
                }
            } else {
                var a = $("<a>")
                a.attr("id", 'img' + img.hash)
                a.attr("href", "/" + params.albumName + "/pano-" + img.hash + ".html")
                a.html('<img alt="" src="/thumb/300w/' + img.hash + '.jpg" srcset="/thumb/600w/' + img.hash + '.jpg 2x" />')

                $(params.galleryPano).show().append(a)
            }
        }

        if (chronoTexts) {
            for (var ti = 0; ti < chronoTexts.length; ti++) {
                var t = chronoTexts[ti]
                $(params.gallery).append("<div class='chrono-text pure-g'><div class='text pure-u-3-5 some-text'>" + t.text + "</div></div>")
            }
        }

        if (typeof result.tracks === 'undefined') {
            result.tracks = [];
        }

        for (var i = 0; i < result.tracks.length; i++) {
            var gpx = result.tracks[i];

            console.log("GPX", gpx);

            if (gpsBounds.maxLat == null) {
                gpsBounds.maxLat = gpx.maxLat
                gpsBounds.minLat = gpx.minLat
                gpsBounds.maxLon = gpx.maxLon
                gpsBounds.minLon = gpx.minLon
            } else {
                if (gpx.maxLon > gpsBounds.maxLon) {
                    gpsBounds.maxLon = gpx.maxLon
                }
                if (gpx.minLon < gpsBounds.minLon) {
                    gpsBounds.minLon = gpx.minLon
                }
                if (gpx.maxLat > gpsBounds.maxLat) {
                    gpsBounds.maxLat = gpx.maxLat
                }
                if (gpx.minLat < gpsBounds.minLat) {
                    gpsBounds.minLat = gpx.minLat
                }
            }
        }

        var lightbox = new PhotoSwipeLightbox({
            gallery: params.gallery,
            children: 'a.image',
            pswpModule: PhotoSwipe,
            bgOpacity: 1.0,
        });


        lightbox.on('uiRegister', function () {
            lightbox.pswp.ui.registerElement({
                name: 'tech-details-button',
                ariaLabel: 'Technical details',
                order: 7,
                isButton: true,
                html: '<i title="Technical details" class="camera-icon ctrl-btn"></i>',
                onClick: (event, el) => {
                    var hash = $(pswp.currSlide.data.element).data('hash')
                    var exh = exifHtml[hash]

                    $('#exif-container').html(exh).toggle();

                    console.log(pswp.currSlide, hash, exh)
                }
            });
        });

        if (fullscreenSupported) {
            lightbox.on('uiRegister', function () {
                lightbox.pswp.ui.registerElement({
                    name: 'fullscreen-button',
                    ariaLabel: 'Toggle full screen',
                    order: 8,
                    isButton: true,
                    html: '<i title="Toggle full screen" class="screen-icon ctrl-btn"></i>',
                    onClick: (event, el) => {
                        toggleFullscreen()
                    }
                });
            });
        }

        // Download image button.
        lightbox.on('uiRegister', function () {
            lightbox.pswp.ui.registerElement({
                name: 'download-button',
                order: 9,
                isButton: true,
                tagName: 'a',

                // SVG with outline
                html: {
                    isCustomSVG: true,
                    inner: '<path d="M20.5 14.3 17.1 18V10h-2.2v7.9l-3.4-3.6L10 16l6 6.1 6-6.1ZM23 23H9v2h14Z" id="pswp__icn-download"/>',
                    outlineID: 'pswp__icn-download'
                },

                onInit: (el, pswp) => {
                    el.setAttribute('download', '');
                    el.setAttribute('target', '_blank');
                    el.setAttribute('rel', 'noopener');

                    pswp.on('change', () => {
                        console.log('change');
                        el.href = pswp.currSlide.data.src;
                    });
                }
            });
        });

        new PhotoSwipeDynamicCaption(lightbox, {
            mobileLayoutBreakpoint: 700,
            type: 'aside',
        });

        var currentImage = {
            album: params.albumName
        }

        lightbox.on('contentResize', ({content, width, height}) => {
            if (width > currentImage.w) {
                currentImage.mw = width
                currentImage.mh = height
            }
        });
        lightbox.on('contentActivate', ({content}) => {
            $('#exif-container').hide();

            if (currentImage.img) {
                currentImage.time = Date.now() - currentImage.time;
                collectStats(currentImage);
            }

            currentImage.img = $(content.data.element).data('hash')
            currentImage.time = Date.now();
            currentImage.w = content.displayedImageWidth
            currentImage.h = content.displayedImageHeight
        });

        lightbox.on('close', () => {
            $('#exif-container').hide();
            if (currentImage.img) {
                currentImage.time = Date.now() - currentImage.time;
                collectStats(currentImage);
                currentImage.img = ""
            }
        });

        lightbox.init();

        window.openByHash = function (hash) {
            // console.log("openByHash", hash, idxByHash[hash])
            lightbox.loadAndOpen(idxByHash[hash], {gallery: document.querySelector(params.gallery)});
        }

        var imgHash = window.location.pathname.match(/photo-(.+)\.html/)

        if (imgHash !== null) {
            imgHash = imgHash[1]
        } else {
            imgHash = window.location.hash.substring(1);
        }

        // console.log("imgHash", imgHash)

        if (imgHash !== "") {
            if (idxByHash[imgHash] !== undefined) {
                window.openByHash(imgHash)
            }
        }

        lightbox.on('change', function () {
            // console.log("change", lightbox.pswp, lightbox.pswp.currIndex, hashByIdx[lightbox.pswp.currIndex])
            history.pushState("", document.title, "/" + params.albumName + "/photo-" + hashByIdx[lightbox.pswp.currIndex] + ".html");
        })

        lightbox.on('close', function () {
            history.pushState("", document.title, originalPath);
            exitFullscreen();
        })

        var thumbSize = "200h"
        if (window.devicePixelRatio > 1) {
            thumbSize = "400h"
        }

        if (params.showMap && gpsBounds.minLat !== null) {
            console.log("GPS BOUNDS", gpsBounds);

            var overlayMaps = {};

            $(params.gallery).append('<div id="map"></div>')

            $('#map').show()
            var map = L.map('map', {
                fullscreenControl: true,
                scrollWheelZoom: false
            }).fitBounds([
                [gpsBounds.minLat, gpsBounds.minLon],
                [gpsBounds.maxLat, gpsBounds.maxLon]
            ]);
            L.tileLayer(params.mapTiles, {
                maxZoom: 19,
                // detectRetina: true,
                attribution: params.mapAttribution
            }).addTo(map);

            L.control.scale().addTo(map);

            var images = []
            for (var i = 0; i < gpsMarkers.length; i++) {
                var img = gpsMarkers[i]
                var m = img.gps
                var w = 300
                if (img.height > img.width) {
                    w = Math.round(w * img.width / img.height)
                }
                var text = '<p>Pos: ' + m.latitude.toFixed(6) + ',' + m.longitude.toFixed(6) + ', Alt: ' + m.altitude + 'm</p>'
                if (img.description) {
                    text += '<div>' + img.description + '</div>'
                }

                images.push(L.marker([m.latitude, m.longitude],
                    {
                        icon: L.icon({
                            iconUrl: '/thumb/' + thumbSize + '/' + m.hash + '.jpg',
                            iconSize: [40],
                            className: 'image-marker'
                        })
                    })
                    .bindPopup(
                        L.popup()
                            .setContent(
                                text +
                                '<a href="#" onclick="openByHash(\'' + m.hash + '\');return false">' +
                                '<img style="width: ' + w + 'px" src="/thumb/200h/' + m.hash + '.jpg" srcset="/thumb/400h/' + m.hash + '.jpg 2x" /></a>'
                            )
                    )
                    .addTo(map)
                )
            }

            overlayMaps["Photos"] = L.layerGroup(images);

            /////////////////////////
            // GPX rendering.

            function getDarkColor() {
                var color = '#';
                for (var i = 0; i < 6; i++) {
                    color += Math.floor(Math.random() * 12).toString(16);
                }
                return color;
            }

            var toRad = Math.PI / 180

            // distance returns 2D distance between two points in meters.
            function distance(lat1, lon1, lat2, lon2) {
                var dLat = toRad * (lat1 - lat2)
                var dLon = toRad * (lon1 - lon2)

                var a = Math.pow(Math.sin(dLat / 2), 2) + Math.pow(Math.sin(dLon / 2), 2) * Math.cos(toRad * lat1) * Math.cos(toRad * lat2)
                var c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))

                return 6371000 * c
            }

            function gpxPopupHandler(name) {
                return function (e) {
                    console.log(e)

                    var popup = e.popup;

                    var feat = e.layer.feature
                    console.log("feat", feat)

                    var res = name + "<br />"
                    if (feat.properties.desc) {
                        res += feat.properties.desc + "<br />"
                    }

                    if (feat.properties.name) {
                        res += feat.properties.name + "<br />"
                    }

                    if (feat.geometry.type === "Point") {

                        res += feat.geometry.coordinates[1].toFixed(8) + ", " + feat.geometry.coordinates[0].toFixed(8)
                        res += '<br /><a class="google-maps" href="https://maps.google.com/maps?q=loc:' +
                            feat.geometry.coordinates[1].toFixed(8) + ',' + feat.geometry.coordinates[0].toFixed(8) + '">google maps</a>'
                        res += '<br /><a class="apple-maps" href="https://maps.apple.com/?ll=' +
                            feat.geometry.coordinates[1].toFixed(8) + ',' + feat.geometry.coordinates[0].toFixed(8) + '">apple maps</a>'

                        popup.setContent(res);
                        return
                    }

                    var points = e.layer._latlngs || []

                    var lat = popup.getLatLng().lat
                    var lon = popup.getLatLng().lng

                    res += lat.toFixed(8) + ", " + lon.toFixed(8)

                    if (typeof e.layer.feature.properties.coordTimes !== 'undefined') {
                        var shortest = null
                        var ptIdx = []
                        for (var i = 0; i < points.length; i++) {
                            var pt = points[i]

                            if (pt instanceof Array) {
                                for (var j = 0; j < pt.length; j++) {
                                    var pt2 = pt[j]

                                    var dist = distance(lat, lon, pt2.lat, pt2.lng)

                                    if (shortest === null || shortest > dist) {
                                        shortest = dist
                                        ptIdx = [i, j]
                                    }
                                }

                                continue
                            }

                            var dist = distance(lat, lon, pt.lat, pt.lng)

                            if (shortest === null || shortest > dist) {
                                shortest = dist
                                ptIdx = [i]
                            }
                        }

                        var ts

                        if (ptIdx.length == 1) {
                            ts = e.layer.feature.properties.coordTimes[ptIdx[0]]
                        }

                        if (ptIdx.length == 2) {
                            ts = e.layer.feature.properties.coordTimes[ptIdx[0]][ptIdx[1]]
                        }

                        res +=
                            '<br />' + ts + " (" + Math.round(shortest) + "m away)"
                    }

                    res += '<br /><a class="google-maps" href="https://maps.google.com/maps?q=loc:' +
                        lat.toFixed(8) + ',' + lon.toFixed(8) + '">google maps</a>'
                    res += '<br /><a class="apple-maps" href="https://maps.apple.com/?ll=' +
                        lat.toFixed(8) + ',' + lon.toFixed(8) + '">apple maps</a>'

                    popup.setContent(res);
                }
            }

            for (var i = 0; i < result.tracks.length; i++) {
                var customLayer = L.geoJson(null, {
                    style: function () {
                        return {color: getDarkColor()};
                    }
                });

                var gpx = result.tracks[i];

                var gpxLayer = omnivore.gpx('/track/' + gpx.hash + '.gpx', null, customLayer);

                gpxLayer.bindPopup(gpx.name).addTo(map);
                gpxLayer.on('popupopen', gpxPopupHandler(gpx.name));

                overlayMaps[gpx.name] = gpxLayer
            }

            L.control.layers({}, overlayMaps).addTo(map);
            L.control.locate({}).addTo(map);
        }
    }


    if (params.albumData != null) {
        renderAlbum(params.albumData)
    } else {
        client.getAlbumContents({
            name: params.albumName
        }, renderAlbum, function (error) {
            alert("Bad request: " + error.error)
        }, function (error) {
            alert("Failed: " + error.error)
        })
    }
}