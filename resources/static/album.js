var lastBlur = null;
var unfocused = 0;
var currentImage = {
    album: "", // Album name.
    img: "" // Image hash.
};

window.openByHashRegistry = window.openByHashRegistry || {}
window.openByHashInGallery = window.openByHashInGallery || function (galleryKey, hash) {
    var opener = window.openByHashRegistry[galleryKey]
    if (opener) {
        opener(hash)
    }
};

(function () {
    document.addEventListener('keyup', function (e) {
        if (e.ctrlKey && e.key === "x") { // Ctrl+X: delete image.
            if (currentImage.img) {
                removeImage(currentImage.album, currentImage.img);
            }
        }

        if ((e.key === "s" || e.key === "S") && !e.ctrlKey && !e.metaKey && !e.altKey) { // S: toggle selection of maximized image, when in select mode.
            var tag = e.target && e.target.tagName
            if (tag !== 'INPUT' && tag !== 'TEXTAREA' &&
                document.documentElement.classList.contains('image-select-mode') && currentImage.img) {
                setImageHashSelected(currentImage.img, !isImageHashSelected(currentImage.img))
                refreshImageSelectCheckboxes()
                updateSelectionWidgets()
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
 * @property {String} collabKey - optional collaborator key to remove images
 * @property {Boolean} showAISays - show AI says in image view
 * @property {Boolean} showEXIFPreview - show EXIF preview in image view
 * @property {Boolean} preRendered - server rendered HTML exists for images
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

    var galleryKey = params.gallery

    if (!params.galleryPano) {
        params.galleryPano = "#gallery-pano"
    }

    var galleryKeyJS = galleryKey.replace(/\\/g, "\\\\").replace(/'/g, "\\'")

    if (!params.baseUrl) {
        params.baseUrl = "/" + params.albumName
    }

    if (params.baseUrl === "/" && window.location.pathname !== "/") {
        params.baseUrl = window.location.pathname + window.location.search
    }

    /**
     * @type {Array<PhotoImage>}
     */
    var gpsMarkers = []
    var client = new Backend('');

    var thumbBase = params.thumbBaseUrl
    if (!thumbBase) {
        thumbBase = "/thumb"
    }

    var imageBase = params.imageBaseUrl
    if (!imageBase) {
        imageBase = "/image"
    }

    function selectCheckboxHTML(hash) {
        return '<input type="checkbox" class="img-select-checkbox" data-hash="' + hash + '" onclick="toggleImageSelect(this, event)" />'
    }

    function classicThumbHTML(img, landscape, aspectRatio) {
        return selectCheckboxHTML(img.hash) +
            '<canvas id="bh-' + img.hash + '" width="32" height="32"></canvas>' +
            '<img alt="photo" src="' + thumbBase + '/200h/' + img.hash + '.jpg" srcset="' + thumbBase + '/400h/' + img.hash + '.jpg ' + Math.round(400 * aspectRatio) + 'w, ' + thumbBase + '/300w/' + img.hash + '.jpg 300w, ' + thumbBase + '/600w/' + img.hash + '.jpg 600w" />'
    }

    function spriteThumbStyle(spriteSheet, width, height, offsetY, backgroundWidth, backgroundHeight) {
        var oneX = "/thumb-sprite/" + spriteSheet.chunk_1x + ".jpg"
        var backgroundImage = "url('" + oneX + "')"

        if (window.CSS && CSS.supports && CSS.supports("background-image", "image-set(url('" + oneX + "') 1x, url('/x.jpg') 2x)")) {
            backgroundImage = "image-set(url('" + oneX + "') 1x, url('/thumb-sprite/" + spriteSheet.chunk_2x + ".jpg') 2x)"
        }

        var scale = 1
        if (backgroundWidth > 0 && width > 0) {
            scale = width / backgroundWidth
        }

        return {
            width: width + "px",
            height: height + "px",
            backgroundImage: backgroundImage,
            backgroundPosition: "0 -" + Math.round(offsetY * scale) + "px",
            backgroundRepeat: "no-repeat",
            backgroundSize: Math.round(backgroundWidth * scale) + "px " + Math.round(backgroundHeight * scale) + "px"
        }
    }

    function applySpriteStyle(spriteEl, spriteSheet, width, height, offsetY, backgroundWidth, backgroundHeight) {
        var style = spriteThumbStyle(spriteSheet, width, height, offsetY, backgroundWidth, backgroundHeight)

        spriteEl.css(style)
    }

    function spriteThumbHTML(img, landscape, sprite) {
        return selectCheckboxHTML(img.hash) +
            '<canvas id="bh-' + img.hash + '" width="32" height="32"></canvas>' +
            '<span class="thumb-sprite"></span>'
    }

    function updateResponsiveSpriteThumbs() {
        var mobileFullWidthThumbs = window.matchMedia && window.matchMedia("(max-width: 576px)").matches

        $(params.gallery).find(".thumb-sprite").each(function () {
            var sprite = $(this)
            var thumb = sprite.closest(".thumb")
            var canvas = thumb.find("canvas")
            var hash = sprite.closest("a.image").attr("data-hash")
            var spriteMeta = params.thumbSprites && hash ? params.thumbSprites[hash] : null
            var spriteSheet = spriteMeta && params.spriteSheets ? params.spriteSheets[spriteMeta.sheet] : null
            if (!spriteMeta || !spriteSheet) {
                return
            }

            var baseWidth = spriteMeta.width || 0
            var baseHeight = spriteMeta.height || 0
            var offsetY = spriteMeta.offset_y || 0
            var bgWidth = spriteMeta.background_width || 0
            var bgHeight = spriteMeta.background_height || 0

            if (!baseWidth || !baseHeight || !bgWidth || !bgHeight) {
                return
            }

            if (mobileFullWidthThumbs) {
                var scaledWidth = thumb.width()
                if (!scaledWidth) {
                    return
                }

                var scale = scaledWidth / baseWidth
                var scaledHeight = Math.round(baseHeight * scale)

                applySpriteStyle(sprite, spriteSheet, scaledWidth, scaledHeight, offsetY, bgWidth, bgHeight)

                canvas.css({
                    width: scaledWidth + "px",
                    height: scaledHeight + "px"
                })
            } else {
                applySpriteStyle(sprite, spriteSheet, baseWidth, baseHeight, offsetY, bgWidth, bgHeight)

                canvas.css({
                    width: "",
                    height: ""
                })
            }
        })
    }

    function markerSpriteStyle(markerSprite) {
        var markerSheet = params.spriteSheets[markerSprite.sheet]
        if (!markerSheet) {
            return ''
        }

        var oneX = "/thumb-sprite/" + markerSheet.chunk_1x + ".jpg"
        var twoX = "/thumb-sprite/" + markerSheet.chunk_2x + ".jpg"

        return 'width:' + markerSprite.width + 'px;height:' + markerSprite.height + 'px;' +
            'background-image:url(\'' + oneX + '\');' +
            'background-image:-webkit-image-set(url(\'' + oneX + '\') 1x, url(\'' + twoX + '\') 2x);' +
            'background-image:image-set(url(\'' + oneX + '\') 1x, url(\'' + twoX + '\') 2x);' +
            'background-position:0 -' + markerSprite.offset_y + 'px;' +
            'background-repeat:no-repeat;' +
            'background-size:' + markerSprite.background_width + 'px ' + markerSprite.background_height + 'px;' +
            'display:block;border-radius:20%;'
    }

    function markerIconFor(hash, fallbackUrl, fallbackSize) {
        var markerSprite = params.markerSprites[hash]
        if (!markerSprite) {
            return L.icon({
                iconUrl: fallbackUrl,
                iconSize: [fallbackSize],
                className: 'image-marker'
            })
        }

        return L.divIcon({
            className: 'map-marker-icon',
            iconSize: [markerSprite.width, markerSprite.height],
            html: '<span class="image-marker" style="' + markerSpriteStyle(markerSprite) + '"></span>'
        })
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
        var thumbSprites = result.thumb_sprites || {}
        var markerSprites = result.marker_sprites || {}
        var spriteSheets = result.sprite_sheets || {}
        params.thumbSprites = thumbSprites
        params.markerSprites = markerSprites
        params.spriteSheets = spriteSheets

        if (typeof result.images === 'undefined') {
            result.images = [];
        }

        var albumSettings = result.album.settings

        // textsBeforeHash/trailingTexts come from the server-computed timeline (already in final
        // display order, already respecting NewestFirst/pagination/whatever ordering exists) — so
        // this renderer never has to compare timestamps or branch on newest_first itself, it just
        // inserts each text where the server says it goes.
        var textsBeforeHash = {}
        var trailingTexts = []
        if (Array.isArray(result.timeline)) {
            var pendingTexts = []
            for (var tli = 0; tli < result.timeline.length; tli++) {
                var tlItem = result.timeline[tli]
                if (tlItem.image) {
                    textsBeforeHash[tlItem.image] = pendingTexts
                    pendingTexts = []
                } else if (tlItem.text) {
                    pendingTexts.push(tlItem)
                }
            }
            trailingTexts = pendingTexts
        }

        var exifHtml = {}

        for (var i = 0; i < result.images.length; i++) {
            var img = result.images[i]

            // Image is likely still being indexed.
            if (!img.blur_hash) {
                continue
            }

            var existing = null
            if (params.preRendered) {
                existing = $("#img" + img.hash)
                if (existing.length === 0) {
                    existing = null
                }
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

                var a = existing
                if (!a) {
                    a = $("<a>")
                    a.attr("id", 'img' + img.hash)
                    a.attr("data-hash", img.hash)
                }

                if (!existing) {
                    if (i < 4) {
                        a.attr("class", "image img" + i + landscape)
                    } else {
                        a.attr("class", "image" + landscape)
                    }
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
                        srcSet += ", " + imageBase + "/" + img.hash + ".jpg " + img.width + "w"
                    }

                    a.attr("data-pswp-width", img.width)
                    a.attr("data-pswp-height", img.height)
                }
                a.attr("data-pswp-srcset", srcSet)
                a.attr("data-ts", img.utime)

                var img_description = '<a title="Edit details" class="control-panel ctrl-btn edit-icon" href="/edit/image/' + img.hash + '.html"></a>'
                if (result.album.name !== featured) {
                    img_description += '<a title="Add to featured" class="control-panel ctrl-btn star-icon" href="#" onclick="addToFeatured(\'' + img.hash + '\');return false"></a>'
                }
                if (params.collabKey) {
                    img_description += '<a title="Remove from album" class="ctrl-btn trash-icon" href="#" onclick="removeImage(\'' + params.albumName + '\',\'' + img.hash + '\',\'' + params.collabKey + '\');return false"></a>'
                } else {
                    img_description += '<a title="Remove from album" class="control-panel ctrl-btn trash-icon" href="#" onclick="removeImage(\'' + params.albumName + '\',\'' + img.hash + '\');return false"></a>'
                }
                var controls_description = img_description

                var exif = {}
                if (typeof img.exif !== "undefined") {
                    exif = img.exif
                }


                if (!params.preRendered) {
                    var dueTexts = textsBeforeHash[img.hash] || []

                    for (var ti = 0; ti < dueTexts.length; ti++) {
                        var div = $("<div class='chrono-text'><div class='text some-text'>" + dueTexts[ti].text + "</div></div>")

                        if (existing) {
                            a.before(div)
                        } else {
                            $(params.gallery).append(div)
                        }
                    }
                }

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

                if (img.exif) {
                    img_description += '<div class="quick-exif">' +
                        img.exif.camera_model +
                        ' ' + img.exif.lens_model +
                        ' @ ' + (img.exif.focal_length ? (img.exif.focal_length + 'mm') : '') +
                        ' ' + (img.exif.exposure_time ? img.exif.exposure_time + 's' : '') +
                        ' ' + (img.exif.f_number ? 'f/' + img.exif.f_number : '') +
                        ' ' + (img.exif.iso_speed ? 'iso' + img.exif.iso_speed : '') +
                        '</div>'
                }


                if (typeof img.meta !== "undefined") {
                    img_description += '<div class="control-panel">'
                    var cl = img.meta.image_classification

                    if (img.meta.faces && img.meta.faces.length > 0) {
                        img_description += img.meta.faces.length + ' face(s)<br/>'
                    }

                    img_description += '</div>'

                    var llmDescFound = false
                    if (params.showAISays) {
                        for (var ci in img.meta.image_descriptions) {
                            var l = img.meta.image_descriptions[ci]

                            if (!l.text) {
                                console.log("INVALID L", l)

                                continue
                            }

                            llmDescFound = true

                            img_description += '<div class="ai-says" style="margin-top: 20px"><span title="I don\'t always talk bullshit, but when I do, I\'m confident" class="icon-link robot-icon"></span><em title="' + l.model + ': ' + l.prompt + '">AI says:</em><br/>' + l.text.split("\n").join("<br />") + '</div>'
                            break
                        }

                        if (!llmDescFound) {
                            for (var ci in cl) {
                                var l = cl[ci]

                                if (l.model === "cf-uform-gen2" && !llmDescFound) {
                                    img_description += '<div class="ai-says" style="margin-top: 20px"><span title="I don\'t always talk bullshit, but when I do, I\'m confident" class="icon-link robot-icon"></span><em>AI says:</em><br/>' + l.text.split("\n").join("<br />") + '</div>'
                                    break
                                }
                            }
                        }
                    }

                    if (img.meta.geo_label) {
                        img_description += '<div style="margin-top: 20px"><span class="icon-link location-icon"></span>' + img.meta.geo_label + '</div>'
                    }
                }

                var aspectRatio = img.width / img.height

                if (existing) {
                    var thumbEl = a.find(".thumb")
                    var spriteEl = a.find(".thumb-sprite")
                    var hasSpriteBackground = spriteEl.length > 0
                    if (!hasSpriteBackground && a.find("#bh-" + img.hash).length === 0) {
                        a.find(".thumb").prepend('<canvas id="bh-' + img.hash + '" width="32" height="32"></canvas>')
                    }
                    var caption = $('.pswp-caption-content[data-hash="' + img.hash + '"]')
                    if (caption.length === 0) {
                        a.after('<div class="pswp-caption-content" data-hash="' + img.hash + '" style="display: none">' + controls_description + '</div>')
                    } else if (caption.find('.edit-icon').length === 0) {
                        caption.prepend(controls_description)
                    }
                } else {
                    var spriteThumb = thumbSprites[img.hash]
                    var thumbHTML = classicThumbHTML(img, landscape, aspectRatio)
                    if (spriteThumb) {
                        thumbHTML = spriteThumbHTML(img, landscape, spriteThumb)
                    }

                    a.html('<div class="thumb' + landscape + '">' + thumbHTML + '</div>')

                    $(params.gallery).append(a)
                    $(params.gallery).append('<div class="pswp-caption-content" data-hash="' + img.hash + '" style="display: none">' + img_description + '</div>')
                }
                if (typeof img.blur_hash !== "undefined" && document.getElementById('bh-' + img.hash)) {
                    blurHash(img.blur_hash, document.getElementById('bh-' + img.hash))
                }
            } else {
                if (existing) {
                    $(params.galleryPano).show()
                } else {
                    var a = $("<a>")
                    a.attr("id", 'img' + img.hash)
                    a.attr("href", "/" + params.albumName + "/pano-" + img.hash + ".html")
                    a.html('<img alt="" src="' + thumbBase + '/300w/' + img.hash + '.jpg" srcset="' + thumbBase + '/600w/' + img.hash + '.jpg 2x" />')

                    $(params.galleryPano).show().append(a)
                }
            }
        }

        updateResponsiveSpriteThumbs()
        refreshImageSelectCheckboxes()
        updateSelectionWidgets()


        if (!params.preRendered) {
            for (var ti = 0; ti < trailingTexts.length; ti++) {
                $(params.gallery).append("<div class='chrono-text'><div class='text some-text'>" + trailingTexts[ti].text + "</div></div>")
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
            thumbSelector: '.thumb-sprite',
            pswpModule: PhotoSwipe,
            bgOpacity: 1.0
        });

        lightbox.addFilter('domItemData', (itemData, element, linkEl) => {
            var sourceEl = linkEl || element
            if (!sourceEl) {
                return itemData
            }

            var hash = sourceEl.getAttribute('data-hash')
            if (!hash) {
                return itemData
            }

            itemData.msrc = thumbBase + '/300w/' + hash + '.jpg'
            itemData.thumbCropped = true

            return itemData
        });

        lightbox.addFilter('thumbEl', (thumbnail, itemData) => {
            if (thumbnail) {
                return thumbnail
            }

            if (itemData && itemData.element) {
                return itemData.element.querySelector('.thumb-sprite') || itemData.element.querySelector('.thumb img')
            }

            return thumbnail
        });

        // Add a slideshow to the PhotoSwipe gallery.
        const _slideshowPlugin = new PhotoSwipeSlideshow(lightbox, {
            defaultDelayMs: 5000, // 5 seconds
            restartOnSlideChange: true,
            progressBarPosition: "top",
            autoHideProgressBar: false
        });

        // Plugin to display video.
        const _videoPlugin = new PhotoSwipeVideoPlugin(lightbox, {});

        // Hide the PhotoSwipe UI after some time of inactivity.
        //const _autoHideUI = new PhotoSwipeAutoHideUI(lightbox, {});


        // Selection checkbox in the top bar, so it stays visible in full screen.
        lightbox.on('uiRegister', function () {
            lightbox.pswp.ui.registerElement({
                name: 'select-checkbox',
                order: 7.5, // Right of tech details (7, after the preloader's margin-right: auto), before fullscreen (8).
                isButton: false,
                html: '<input type="checkbox" title="Select" class="img-select-checkbox img-select-checkbox-caption" onclick="toggleImageSelect(this, event)" />',
                onInit: (el, pswp) => {
                    var cb = el.querySelector('input')
                    pswp.on('change', () => {
                        cb.setAttribute('data-hash', $(pswp.currSlide.data.element).data('hash'))
                        refreshImageSelectCheckboxes()
                    });
                }
            });
        });

        lightbox.on('uiRegister', function () {
            lightbox.pswp.ui.registerElement({
                name: 'tech-details-button',
                ariaLabel: 'Technical details',
                order: 7,
                isButton: true,
                title: 'Technical details',
                // Camera icon (https://www.iconfinder.com/icons/211634/camera_icon) scaled from 512 to 32 viewBox,
                // outlined like built-in PhotoSwipe icons.
                html: {
                    isCustomSVG: true,
                    inner: '<g id="pswp__icn-camera" transform="translate(6.3 6.3) scale(0.038)"><path vector-effect="non-scaling-stroke" d="M430.4,147h-67.5l-40.4-40.8c0,0-0.2-0.2-0.3-0.2l-0.2-0.2v0c-6-6-14.1-9.8-23.3-9.8h-84c-9.8,0-18.5,4.2-24.6,10.9l0,0.1l-39.5,40H81.6C63,147,48,161.6,48,180.2v202.1c0,18.6,15,33.7,33.6,33.7h348.8c18.5,0,33.6-15.1,33.6-33.7V180.2C464,161.6,448.9,147,430.4,147z M256,365.5c-50.9,0-92.4-41.6-92.4-92.6c0-51.1,41.5-92.6,92.4-92.6c51,0,92.4,41.5,92.4,92.6C348.4,323.9,307,365.5,256,365.5z M424.1,200.5c-7.7,0-14-6.3-14-14.1s6.3-14.1,14-14.1c7.7,0,14,6.3,14,14.1S431.8,200.5,424.1,200.5z"/><path vector-effect="non-scaling-stroke" d="M256,202.9c-38.6,0-69.8,31.3-69.8,70c0,38.6,31.2,70,69.8,70c38.5,0,69.8-31.3,69.8-70C325.8,234.2,294.5,202.9,256,202.9z"/></g>',
                    outlineID: 'pswp__icn-camera'
                },
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
                title: 'Toggle full screen',
                // Monitor icon (https://www.iconfinder.com/icons/211777/monitor_icon), outlined like built-in PhotoSwipe icons.
                html: {
                    isCustomSVG: true,
                    inner: '<g id="pswp__icn-fullscreen" transform="translate(7.2 7.2) scale(0.55)"><path vector-effect="non-scaling-stroke" d="M28,2h-6c-1.104,0-2,0.896-2,2s0.896,2,2,2h1.2l-4.6,4.601C18.28,10.921,18,11.344,18,12c0,1.094,0.859,2,2,2c0.641,0,1.049-0.248,1.4-0.6L26,8.8V10c0,1.104,0.896,2,2,2s2-0.896,2-2V4C30,2.896,29.104,2,28,2z M12,18c-0.641,0-1.049,0.248-1.4,0.6L6,23.2V22c0-1.104-0.896-2-2-2s-2,0.896-2,2v6c0,1.104,0.896,2,2,2h6c1.104,0,2-0.896,2-2s-0.896-2-2-2H8.8l4.6-4.601C13.72,21.079,14,20.656,14,20C14,18.906,13.141,18,12,18z"/></g>',
                    outlineID: 'pswp__icn-fullscreen'
                },
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
                        // console.log(pswp.currSlide.data);
                        el.href = pswp.currSlide.data.src.replace('/image/', '/image-dl/');
                    });
                }
            });
        });

        if (visitorData.showExif || params.showEXIFPreview) {
            // Quick EXIF info.
            lightbox.on('uiRegister', function () {
                lightbox.pswp.ui.registerElement({
                    name: 'custom-caption',
                    order: 10,
                    isButton: false,
                    appendTo: 'root',
                    html: '',
                    onInit: (el, pswp) => {
                        lightbox.pswp.on('change', () => {
                            const currSlideElement = lightbox.pswp.currSlide.data.element;
                            let quickExifHTML = '';

                            if (currSlideElement) {
                                const hash = currSlideElement.getAttribute('data-hash');
                                if (hash) {
                                    const hiddenCaption = document.querySelector('.pswp-caption-content[data-hash="' + hash + '"] .quick-exif');
                                    if (hiddenCaption) {
                                        quickExifHTML = hiddenCaption.innerHTML;
                                    }
                                }
                            }

                            el.innerHTML = quickExifHTML;
                            el.style.display = quickExifHTML.trim() ? '' : 'none';
                        });
                    }
                });
            });
        }

        new PhotoSwipeDynamicCaption(lightbox, {
            mobileLayoutBreakpoint: 700,
            type: 'aside',
            captionContent: (slide) => {
                if (!slide || !slide.data || !slide.data.element) {
                    return '';
                }
                const hash = slide.data.element.getAttribute('data-hash');
                if (!hash) {
                    return '';
                }
                const captionEl = document.querySelector('.pswp-caption-content[data-hash="' + hash + '"]');
                return captionEl ? captionEl.innerHTML : '';
            }
        });

        lightbox.on('contentResize', ({content, width, height}) => {
            if (width > currentImage.w) {
                currentImage.mw = width
                currentImage.mh = height
            }
        });
        lightbox.on('contentActivate', ({content}) => {
            $('#exif-container').hide();

            if (currentImage.img) {
                currentImage.album = params.albumName
                currentImage.time = Date.now() - currentImage.time - unfocused;
                collectStats(currentImage);
            }

            currentImage.album = params.albumName
            currentImage.img = $(content.data.element).data('hash')
            currentImage.time = Date.now() - unfocused;
            currentImage.w = content.displayedImageWidth
            currentImage.h = content.displayedImageHeight
        });

        lightbox.on('close', () => {
            $('#exif-container').hide();
            if (currentImage.img) {
                currentImage.time = Date.now() - currentImage.time - unfocused;
                currentImage.album = params.albumName

                collectStats(currentImage);
                currentImage.img = ""
                currentImage.album = ""
            }
        });

        lightbox.init();

        var openByHash = function (hash) {
            // console.log("openByHash", hash, idxByHash[hash])
            lightbox.loadAndOpen(idxByHash[hash], {gallery: document.querySelector(params.gallery)});
        }
        window.openByHashRegistry[galleryKey] = openByHash

        var imgHash = window.location.pathname.match(/photo-(.+)\.html/)

        if (imgHash !== null) {
            imgHash = imgHash[1]
        } else {
            imgHash = window.location.hash.substring(1);
        }

        // console.log("imgHash", imgHash)

        if (imgHash !== "") {
            if (idxByHash[imgHash] !== undefined) {
                openByHash(imgHash)
            }
        }

        function syncCurrentImageAddress() {
            if (!lightbox.pswp) {
                return
            }

            var hash = hashByIdx[lightbox.pswp.currIndex]
            if (!hash) {
                return
            }

            var nextPath = "/" + params.albumName + "/photo-" + hash + ".html"
            if (window.location.pathname === nextPath) {
                return
            }

            history.replaceState("", document.title, nextPath);
        }

        function restoreAlbumAddress() {
            if (window.location.pathname === params.baseUrl) {
                return
            }

            history.replaceState("", document.title, params.baseUrl);
        }

        lightbox.on('change', function () {
            if (fullscreenEnabled) {
                return
            }

            syncCurrentImageAddress()
        })

        window.syncGalleryAddressAfterFullscreen = syncCurrentImageAddress

        lightbox.on('close', function () {
            restoreAlbumAddress()
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
                        icon: markerIconFor(m.hash, thumbBase + '/' + thumbSize + '/' + m.hash + '.jpg', 40)
                    })
                    .bindPopup(
                        L.popup()
                            .setContent(
                                text +
                                '<a href="#" onclick="openByHashInGallery(\'' + galleryKeyJS + '\', \'' + m.hash + '\');return false">' +
                                '<img style="width: ' + w + 'px" src="' + thumbBase + '/200h/' + m.hash + '.jpg" srcset="' + thumbBase + '/400h/' + m.hash + '.jpg 2x" /></a>'
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

    window.addEventListener('resize', updateResponsiveSpriteThumbs)
}
