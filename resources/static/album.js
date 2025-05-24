var lastBlur = null;
var unfocused = 0;
var currentImage = {
    album: "", // Album name.
    img: "" // Image hash.
};

(function () {
    document.addEventListener('keyup', function (e) {
        if (e.ctrlKey && e.key === "x") { // Ctrl+X: delete image.
            if (currentImage.img) {
                removeImage(currentImage.album, currentImage.img);
            }
        }
        // console.log("key pressed", e)
    }, false);

    window.addEventListener('blur', function () {
        lastBlur = new Date();
    });

    window.addEventListener('focus', function () {
        if (lastBlur) {
            unfocused += new Date() - lastBlur;
        }

        lastBlur = null
    });

    window.addEventListener('scroll', function () {
        if (lastBlur) {
            unfocused += new Date() - lastBlur;
        }

        lastBlur = null
    });


    if (screen.width > 576 || screen.width == 0) {
        return
    }

    var styles = `
@media screen and (orientation:portrait) {
    .main {
        margin-left: 0;
        margin-right: 0;
    }
    .thumb, a.image, .chrono-text {
        width: ` + screen.width + `px;
        height: fit-content;
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
 * @property {UsecaseGetAlbumOutput} albumData
 * @property {String} gallery - CSS selector for gallery container
 * @property {String} galleryPano - CSS selector for gallery panoramas container
 * @property {String} baseUrl - base address to set on image close
 * @property {Boolean} enableFavorite - allow favorite pictures
 * @property {String} imageBaseUrl - base address to link to full-res images
 * @property {String} thumbBaseUrl - thumbnail base URL
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

    if (!params.mapTiles) {
        params.mapTiles = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
    }

    if (!params.mapAttribution) {
        params.mapAttribution = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
    }

    if (!params.gallery) {
        params.gallery = "#gallery"
    }

    if (!params.galleryPano) {
        params.galleryPano = "#gallery-pano"
    }

    /**
     * @type {Array<PhotoImage>}
     */
    var gpsMarkers = []
    var client = new Backend('');

    var originalPath = params.baseUrl;
    if (!originalPath) {
        originalPath = window.location.toString();
    }

    var thumbBase = params.thumbBaseUrl
    if (!thumbBase) {
        thumbBase = "/thumb"
    }

    var imageBase = params.imageBaseUrl
    if (!imageBase) {
        imageBase = "/image"
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

                var landscape = ""
                if (img.width >= img.height) {
                    landscape = " landscape"
                } else {
                    landscape = " portrait"
                }

                var a = $("<a>")
                a.attr("id", 'img' + img.hash)
                a.attr("data-hash", img.hash)

                if (i < 4) {
                    a.attr("class", "image img" + i + landscape)
                } else {
                    a.attr("class", "image" + landscape)
                }
                if (hideOriginal) {
                    a.attr("href", "#")
                } else {
                    a.attr("href", imageBase + "/" + img.hash + ".jpg")
                }
                a.attr("target", "_blank")
                a.attr("data-idx", i)

                var srcSet = thumbBase + "/300w/" + img.hash + ".jpg 300w" +
                    ", " + thumbBase + "/600w/" + img.hash + ".jpg 600w" +
                    ", " + thumbBase + "/1200w/" + img.hash + ".jpg 1200w"


                if (!visitorData.lowRes) {
                    srcSet += ", " + thumbBase + "/2400w/" + img.hash + ".jpg 2400w"
                }

                if (img.width > 0 && img.height > 0) {
                    if (!hideOriginal && !visitorData.lowRes) {
                        srcSet += ", "+imageBase+"/" + img.hash + ".jpg " + img.width + "w"
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

                        var div = $("<div data-ts='" + t.time + "' class='chrono-text'><div class='text some-text'>" + t.text + "</div></div>")

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

                    if (img.meta.faces && img.meta.faces.length > 0) {
                        img_description += img.meta.faces.length + ' face(s)<br/>'
                    }

                    img_description += '</div>'

                    var llmDescFound = false
                    for (var ci in img.meta.image_descriptions) {
                        var l = img.meta.image_descriptions[ci]

                        if (!l.text) {
                            console.log("INVALID L", l)

                            continue
                        }

                        llmDescFound = true

                        img_description += '<div class="ai-says" style="margin-top: 20px"><span title="I don\'t always talk bullshit, but when I do, I\'m confident" class="icon-link robot-icon"></span><em title="'+l.model+': ' + l.prompt+'">AI says:</em><br/>' + l.text.split("\n").join("<br />") + '</div>'
                    }

                    for (var ci in cl) {
                        var l = cl[ci]

                        if (l.model === "cf-uform-gen2" && !llmDescFound) {
                            img_description += '<div class="ai-says" style="margin-top: 20px"><span title="I don\'t always talk bullshit, but when I do, I\'m confident" class="icon-link robot-icon"></span><em>AI says:</em><br/>' + l.text.split("\n").join("<br />") + '</div>'
                        }
                    }

                    if (img.meta.geo_label) {
                        img_description += '<div style="margin-top: 20px"><span class="icon-link location-icon"></span>' + img.meta.geo_label + '</div>'
                    }
                }

                var aspectRatio = img.width / img.height

                a.attr("data-pswp-srcset", srcSet)
                a.html('<div class="thumb' + landscape + '">' +
                    '<canvas id="bh-' + img.hash + '" width="32" height="32"></canvas>' +
                    '<img alt="photo" src="'+thumbBase+'/200h/' + img.hash + '.jpg" srcset="'+thumbBase+'/400h/' + img.hash + '.jpg ' + Math.round(400 * aspectRatio) + 'w, '+thumbBase+'/300w/' + img.hash + '.jpg 300w, '+thumbBase+'/600w/' + img.hash + '.jpg 600w" /></div>')
                a.attr("data-ts", img.time)

                a.append('<div class="pswp-caption-content" data-hash="' + img.hash + '" style="display: none">' + img_description + '</div>')

                $(params.gallery).append(a)
                if (typeof img.blur_hash !== "undefined") {
                    blurHash(img.blur_hash, document.getElementById('bh-' + img.hash))
                }
            } else {
                var a = $("<a>")
                a.attr("id", 'img' + img.hash)
                a.attr("href", "/" + params.albumName + "/pano-" + img.hash + ".html")
                a.html('<img alt="" src="'+thumbBase+'/300w/' + img.hash + '.jpg" srcset="'+thumbBase+'/600w/' + img.hash + '.jpg 2x" />')

                $(params.galleryPano).show().append(a)
            }
        }


        if (chronoTexts) {
            for (var ti = 0; ti < chronoTexts.length; ti++) {
                var t = chronoTexts[ti]
                $(params.gallery).append("<div class='chrono-text'><div class='text some-text'>" + t.text + "</div></div>")
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


        if (albumSettings.map_min_lat) {
            gpsBounds.minLat = albumSettings.map_min_lat
        }
        if (albumSettings.map_max_lat) {
            gpsBounds.maxLat = albumSettings.map_max_lat
        }
        if (albumSettings.map_min_lon) {
            gpsBounds.minLon = (albumSettings.map_min_lon)
        }
        if (albumSettings.map_max_lon) {
            gpsBounds.maxLon = albumSettings.map_max_lon
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

        // if (fullscreenSupported) {
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
        // }

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

        currentImage = {
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
                currentImage.time = Date.now() - currentImage.time - unfocused;
                collectStats(currentImage);
            }

            currentImage.img = $(content.data.element).data('hash')
            currentImage.time = Date.now() - unfocused;
            currentImage.w = content.displayedImageWidth
            currentImage.h = content.displayedImageHeight
        });

        lightbox.on('close', () => {
            $('#exif-container').hide();
            if (currentImage.img) {
                currentImage.time = Date.now() - currentImage.time - unfocused;
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
                            iconUrl: thumbBase+'/' + thumbSize + '/' + m.hash + '.jpg',
                            iconSize: [40],
                            className: 'image-marker'
                        })
                    })
                    .bindPopup(
                        L.popup()
                            .setContent(
                                text +
                                '<a href="#" onclick="openByHash(\'' + m.hash + '\');return false">' +
                                '<img style="width: ' + w + 'px" src="'+thumbBase+'/200h/' + m.hash + '.jpg" srcset="'+thumbBase+'/400h/' + m.hash + '.jpg 2x" /></a>'
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
        if (params.enableFavorite) {
            fillFavorite(params.albumData.album.hash)
        }
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
