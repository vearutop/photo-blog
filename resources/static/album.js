function loadAlbum(albumName, mapTiles, mapAttribution) {
    "use strict";

    var gpsBounds = {
        minLat: null,
        maxLat: null,
        minLng: null,
        maxLng: null
    };

    if (mapTiles == "") {
        mapTiles = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
    }

    if (mapAttribution == "") {
        mapAttribution = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
    }

    /**
     * @type {Array<PhotoGps>}
     */
    var gpsMarkers = []
    var client = new Backend('');

    client.getAlbumImagesNameJson({
        name: albumName
    }, function (result) {
        var hashByIdx = {}
        var idxByHash = {}
        var idx = 0

        for (var i = 0; i < result.images.length; i++) {
            var img = result.images[i]

            if (typeof img.gps != 'undefined') {
                gpsMarkers.push(img.gps)
                if (gpsBounds.maxLat == null) {
                    gpsBounds.maxLat = img.gps.latitude
                    gpsBounds.minLat = img.gps.latitude
                    gpsBounds.maxLng = img.gps.longitude
                    gpsBounds.minLng = img.gps.longitude
                } else {
                    if (img.gps.longitude > gpsBounds.maxLng) {
                        gpsBounds.maxLng = img.gps.longitude
                    }
                    if (img.gps.longitude < gpsBounds.minLng) {
                        gpsBounds.minLng = img.gps.longitude
                    }
                    if (img.gps.latitude > gpsBounds.maxLat) {
                        gpsBounds.maxLat = img.gps.latitude
                    }
                    if (img.gps.latitude < gpsBounds.minLat) {
                        gpsBounds.minLat = img.gps.latitude
                    }
                }
            }

            if (typeof img.exif == "undefined" || img.exif.projection_type === "") {
                // console.log("img", i, idx, img.hash, img)

                hashByIdx[idx] = img.hash
                idxByHash[img.hash] = idx
                idx++

                var a = $("<a>")
                a.attr("id", 'img' + img.hash)
                a.attr("class", "image")
                a.attr("href", "/image/" + img.hash + ".jpg")
                a.attr("target", "_blank")
                a.attr("data-idx", i)

                var srcSet = "/thumb/300w/" + img.hash + ".jpg 300w" +
                    ", /thumb/600w/" + img.hash + ".jpg 600w" +
                    ", /thumb/1200w/" + img.hash + ".jpg 1200w" +
                    ", /thumb/2400w/" + img.hash + ".jpg 2400w"

                var containerStyle = ""

                if (img.width > 0 && img.height > 0) {
                    srcSet += ", /image/" + img.hash + ".jpg " + img.width + "w"
                    a.attr("data-pswp-width", img.width)
                    a.attr("data-pswp-height", img.height)
                    // containerStyle = "height:200px;width:"+ Math.round(img.width * 200 / img.height) + "px"
                    containerStyle = "height:200px;width:300px"
                    a.attr("style", containerStyle)
                }

                var img_description =
                    '<a class="control-panel ctrl-btn edit-icon" href="/edit/image/' + img.hash + '.html"></a>'


                if (typeof img.description !== "undefined") {
                    img_description += "<br>" + img.description
                }

                if (typeof img.exif !== "undefined") {
                    img_description += '<a href="#" class="gear-icon ctrl-btn" onclick="$(this).next().toggle();return false;"></a><div class="exif" style="display: none"><table>';

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

                a.attr("data-pswp-srcset", srcSet)
                a.html('<div class="thumb" style="display:inline-block;background:#333;text-align:center;' + containerStyle + '">' +
                    '<canvas id="bh-' + img.hash + '" width="32" height="32"></canvas>' +
                    '<img alt="" src="/thumb/200h/' + img.hash + '.jpg" srcset="/thumb/400h/' + img.hash + '.jpg 2x" /></div>')
                a.find("img").attr("alt", img_description)

                $(".gallery").append(a)
                if (typeof img.blur_hash !== "undefined") {
                    blurHash(img.blur_hash, document.getElementById('bh-' + img.hash))
                }
            } else {
                var a = $("<a>")
                a.attr("id", 'img' + img.hash)
                a.attr("href", "/" + albumName + "/pano-" + img.hash + ".html")
                a.html('<img alt="" src="/thumb/200h/' + img.hash + '.jpg" srcset="/thumb/400h/' + img.hash + '.jpg 2x" />')

                $(".gallery-pano").show().append(a)
            }
        }

        // console.log("idxByHash", idxByHash)
        // console.log("hashByIdx", hashByIdx)

        var lightbox = new PhotoSwipeLightbox({
            gallery: '.gallery',
            children: 'a.image',
            pswpModule: PhotoSwipe
        });

        new PhotoSwipeDynamicCaption(lightbox, {
            mobileLayoutBreakpoint: 700,
            type: 'auto',
            mobileCaptionOverlapRatio: 1
        });

        lightbox.init();

        window.openByHash = function (hash) {
            // console.log("openByHash", hash, idxByHash[hash])
            lightbox.loadAndOpen(idxByHash[hash], {gallery: document.querySelector('.gallery')});
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
            history.pushState("", document.title, "/" + albumName + "/photo-" + hashByIdx[lightbox.pswp.currIndex] + ".html");
        })

        lightbox.on('close', function () {
            history.pushState("", document.title, "/" + albumName + "/");
        })

        if (gpsBounds.minLat != null) {
            $(".gallery").append('<div id="map"></div>')

            $('#map').show()
            var map = L.map('map').fitBounds([
                [gpsBounds.minLat, gpsBounds.minLng],
                [gpsBounds.maxLat, gpsBounds.maxLng]
            ]);
            L.tileLayer(mapTiles, {
                maxZoom: 19,
                // detectRetina: true,
                attribution: mapAttribution
            }).addTo(map);

            for (var i = 0; i < gpsMarkers.length; i++) {
                var m = gpsMarkers[i]
                L.marker([m.latitude, m.longitude],
                    {
                        icon: L.icon({
                            iconUrl: '/thumb/300w/' + m.hash + '.jpg',
                            iconSize: [40],
                            className: 'image-marker'
                        })
                    })
                    .addTo(map)
                    .bindPopup(
                        L.popup()
                            .setContent(
                                '<p>Pos: ' + m.latitude.toFixed(6) + ',' + m.longitude.toFixed(6) + ', Alt: ' + m.altitude + 'm</p>' +
                                '<a href="#" onclick="openByHash(\'' + m.hash + '\');return false">' +
                                '<img style="width: 300px" src="/thumb/300w/' + m.hash + '.jpg" srcset="/thumb/600w/' + m.hash + '.jpg 2x" /></a>'
                            )
                    )
            }
        }
    }, function (error) {

    }, function (error) {

    })
}