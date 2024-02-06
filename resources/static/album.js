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
    } else {
        $('html')[0].requestFullscreen()
    }
}

function exitFullscreen() {
    if (document.fullscreenElement) {
        document.exitFullscreen()
    }
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
 */


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

        var prevImgTime = null

        var fullscreenSupported = false
        if (typeof document.exitFullscreen == 'function') {
            fullscreenSupported = true
        }

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
                if (i < 4) {
                    a.attr("class", "image img" + i)
                } else {
                    a.attr("class", "image")
                }
                if (hideOriginal) {
                    a.attr("href", "#")
                } else {
                    a.attr("href", "/image/" + img.hash + ".jpg")
                }
                a.attr("target", "_blank")
                a.attr("data-idx", i)

                var srcSet = "/thumb/300w/" + img.hash + ".jpg 300w" +
                    ", /thumb/600w/" + img.hash + ".jpg 600w" +
                    ", /thumb/1200w/" + img.hash + ".jpg 1200w" +
                    ", /thumb/2400w/" + img.hash + ".jpg 2400w"


                if (img.width > 0 && img.height > 0) {
                    if (!hideOriginal) {
                        srcSet += ", /image/" + img.hash + ".jpg " + img.width + "w"
                    }
                    a.attr("data-pswp-width", img.width)
                    a.attr("data-pswp-height", img.height)
                }

                var img_description = '<a title="Edit details" class="control-panel ctrl-btn edit-icon" href="/edit/image/' + img.hash + '.html"></a>'
                if (result.album.name !== featured) {
                    img_description += '<a title="Add to featured" class="control-panel ctrl-btn star-icon" href="#" onclick="addToFeatured(\'' + img.hash + '\');return false"></a>'
                }
                img_description += '<a title="Remove from album" class="control-panel ctrl-btn trash-icon" href="#" onclick="return removeImage(\'' + params.albumName + '\',\'' + img.hash + '\')"></a>'

                if (fullscreenSupported) {
                    img_description += '<a href="#" class="screen-icon ctrl-btn" title="Toggle full screen" onclick="toggleFullscreen();return false;"></a>'
                }

                if (typeof img.exif !== "undefined") {
                    var ts = Date.parse(img.exif.digitized)

                    if (result.album.settings.texts) {
                        for (var ti = 0; ti < result.album.settings.texts.length; ti++) {
                            var t = result.album.settings.texts[ti]

                            var tt = Date.parse(t.time)

                            if (tt > ts) {
                                continue
                            }

                            if (prevImgTime !== null && tt < prevImgTime) {
                                continue
                            }

                            $(params.gallery).append("<div class='chrono-text pure-g'><div class='text pure-u-3-5 some-text'>" + t.text + "</div></div>")
                        }
                    }

                    prevImgTime = ts

                    img_description += '<a href="#" class="camera-icon ctrl-btn" title="Technical details" onclick="$(this).next().toggle();return false;"></a><div class="exif" style="display: none"><table>';

                    var exif = img.exif

                    exif["file_name"] = img.name

                    for (var k in exif) {
                        if (k === "hash" || k === "exposure_time_sec" || k === "created_at") {
                            continue;
                        }

                        var v = exif[k]

                        if (!v) {
                            continue;
                        }

                        img_description += "<tr><th>" + k + "</th><td>" + v + "</td>";
                    }
                    img_description += '</table></div>';
                }

                if (img.description) {
                    img_description += '<div>' + img.description + '</div>';
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

        new PhotoSwipeDynamicCaption(lightbox, {
            mobileLayoutBreakpoint: 700,
            type: 'aside',
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
            }).fitBounds([
                [gpsBounds.minLat, gpsBounds.minLon],
                [gpsBounds.maxLat, gpsBounds.maxLon]
            ]);
            L.tileLayer(params.mapTiles, {
                maxZoom: 19,
                // detectRetina: true,
                attribution: params.mapAttribution
            }).addTo(map);

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

                    if (feat.geometry.type === "Point") {
                        res += feat.geometry.coordinates[1].toFixed(8) + ", " + feat.geometry.coordinates[0].toFixed(8)

                        popup.setContent(res);
                        return
                    }

                    var points = e.layer._latlngs || []

                    var lat = popup.getLatLng().lat
                    var lon = popup.getLatLng().lng

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

                    if (typeof e.layer.feature.properties.coordTimes === 'undefined') {
                        popup.setContent(res);
                    }

                    var ts

                    if (ptIdx.length == 1) {
                        ts = e.layer.feature.properties.coordTimes[ptIdx[0]]
                    }

                    if (ptIdx.length == 2) {
                        ts = e.layer.feature.properties.coordTimes[ptIdx[0]][ptIdx[1]]
                    }

                    res += lat.toFixed(8) + ", " + lon.toFixed(8) + "<br />" +
                        ts + "<br /> (dist to nearest timestamp " + Math.round(shortest) + " meters)"
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