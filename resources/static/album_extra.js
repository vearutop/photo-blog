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

function reindexAlbum(name) {
    var b = new Backend('');
    b.controlIndexAlbum({
        name: name,
    }, function (x) {
        alert("Reindexing started")
    }, function (x) {
        alert("Failed to reindex album: " + x.error)
    })
}


function beforeUploadRequest(req, file, allFiles) {
    console.log("before upload", req, file, allFiles)

    var name = file.data.name;
    var relativePath = file.data.relativePath;

    console.log(name, relativePath)

    const thumbSizes = ["300w", "600w", "1200w", "2400w"]

    if (relativePath) {
        for (var j = 0; j < thumbSizes.length; j++) {
            var t = thumbSizes[j]
            if (relativePath.includes(t)) {
                return;
            }
        }
    }

    var thumbs = [];

    for (var i = 0; i < allFiles.length; i++) {
        var f = allFiles[i].data;

        if (!f.relativePath) {
            continue;
        }

        if (f.name !== name) {
            continue;
        }

        for (var j = 0; j < thumbSizes.length; j++) {
            var t = thumbSizes[j]
            if (f.relativePath.includes(t)) {
                thumbs.push(t)
                break;
            }
        }
    }

    /**
     * @var {XMLHttpRequest} xhr
     */
    var xhr = req.getUnderlyingObject()

    if (thumbs) {
        xhr.setRequestHeader("X-Expect-Thumbnails", thumbs.join(","))
    }
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

function fillFavorite(albumHash) {
    var b = new Backend('');
    b.getFavorite({
        albumHash: albumHash
    }, function (res) {
        var idx = {}
        for (var i in res) {
            idx[res[i]] = true;
        }

        $('.pswp-caption-content').each(function () {
            var h = $(this).data('hash')

            if (idx[h]) {
                $(this).prepend('<a title="Remove from favorites" data-favorite="yes" class="ctrl-btn heart-icon" href="#" onclick="toggleFavorite(\'' + h + '\', this);return false"></a>')
            } else {
                $(this).prepend('<a title="Add to favorites" data-favorite="no" class="ctrl-btn heart-empty-icon" href="#" onclick="toggleFavorite(\'' + h + '\', this);return false"></a>')
            }
        })
    }, function (x) {
        alert("Failed: " + x.error)
    })
}

function toggleFavorite(imageHash, a) {
    console.log(a)

    var b = new Backend('');

    var req = {
        imageHash: imageHash
    }

    if ($(a).data("favorite") === 'yes') {
        b.deleteFavorite({
            imageHash: imageHash
        }, function () {
            $(a).attr("title", "Add to favorite").data('favorite', 'no').removeClass('heart-icon').addClass('heart-empty-icon')
            console.log("favorite deleted")
        })
    } else {
        b.addFavorite(req, function () {
            $(a).attr("title", "Remove from favorites").data('favorite', 'yes').removeClass('heart-empty-icon').addClass('heart-icon')
            console.log("favorite added")
        })
    }
}

// Multi-image selection, backed by localStorage so a selection can span multiple albums.

var IMAGE_SELECT_HASHES_KEY = 'imageSelect.hashes'
var IMAGE_SELECT_MODE_KEY = 'imageSelect.enabled'

function getSelectedImageHashes() {
    try {
        var raw = window.localStorage.getItem(IMAGE_SELECT_HASHES_KEY)
        if (!raw) {
            return []
        }

        var hashes = JSON.parse(raw)
        if (!Array.isArray(hashes)) {
            return []
        }

        return hashes
    } catch (e) {
        return []
    }
}

function isImageHashSelected(hash) {
    return getSelectedImageHashes().indexOf(hash) !== -1
}

function setImageHashSelected(hash, selected) {
    var hashes = getSelectedImageHashes()
    var idx = hashes.indexOf(hash)

    if (selected) {
        if (idx === -1) {
            hashes.push(hash)
            hashes.sort() // Keep a stable order so the "list-" URL is stable regardless of selection order.
        }
    } else if (idx !== -1) {
        hashes.splice(idx, 1)
    }

    window.localStorage.setItem(IMAGE_SELECT_HASHES_KEY, JSON.stringify(hashes))
}

// viewSelectedImagesURL builds a stable "virtual album" URL listing all selected images.
function viewSelectedImagesURL() {
    var hashes = getSelectedImageHashes().slice().sort()
    if (hashes.length === 0) {
        return ""
    }

    return "/list-" + hashes.join(",") + "/"
}

// updateViewSelectedLink shows/hides and updates the "View selected" link in the album title panel.
function updateViewSelectedLink() {
    var link = document.getElementById('view-selected-link')
    if (!link) {
        return
    }

    var count = getSelectedImageHashes().length
    var countEl = document.getElementById('view-selected-count')
    if (countEl) {
        countEl.textContent = count
    }

    if (count > 0) {
        link.href = viewSelectedImagesURL()
        link.style.display = ''
    } else {
        link.style.display = 'none'
    }
}

// refreshImageSelectCheckboxes syncs the checked state of all rendered checkboxes with localStorage.
function refreshImageSelectCheckboxes() {
    var idx = {}
    var hashes = getSelectedImageHashes()
    for (var i = 0; i < hashes.length; i++) {
        idx[hashes[i]] = true
    }

    var checkboxes = document.querySelectorAll('.img-select-checkbox')
    for (var i = 0; i < checkboxes.length; i++) {
        checkboxes[i].checked = !!idx[checkboxes[i].getAttribute('data-hash')]
    }
}

// selectImages shows/hides selection checkboxes on thumbnails and in the maximized view.
// The enabled state is persisted so it survives navigation across album pages.
function selectImages(enable) {
    window.localStorage.setItem(IMAGE_SELECT_MODE_KEY, enable ? '1' : '0')

    if (enable) {
        document.documentElement.classList.add('image-select-mode')
        refreshImageSelectCheckboxes()
    } else {
        document.documentElement.classList.remove('image-select-mode')
    }
}

function toggleImageSelect(checkbox, e) {
    if (e) {
        e.stopPropagation()
    }

    var hash = checkbox.getAttribute('data-hash')
    setImageHashSelected(hash, checkbox.checked)
    refreshImageSelectCheckboxes()
    updateViewSelectedLink()
}

(function () {
    if (window.localStorage.getItem(IMAGE_SELECT_MODE_KEY) === '1') {
        document.documentElement.classList.add('image-select-mode')
    }
})()

function removeImage(albumName, imageHash, collabKey) {
    if (!window.confirm("This photo is about to be removed from the album '" + albumName + "'")) {
        return
    }

    if (!collabKey) {
        collabKey = '';
    }

    var b = new Backend('');
    b.controlRemoveFromAlbum({
        name: albumName,
        hash: imageHash,
        collabKey: collabKey
    }, function (x) {
        $('#img' + imageHash).remove()
        // alert("Photo is removed from the album")
    }, function (x) {
        alert("Failed to remove photo from the album: " + x.error)
    })
}

var fullscreenEnabled = false;

function toggleFullscreen() {
    if (fullscreenEnabled) {
        if (document.fullscreenElement) {
            document.exitFullscreen()
        }
        fullscreenEnabled = false;
        $('html body').css("overflow", "auto").removeClass("fullscreen")
        if (typeof window.syncGalleryAddressAfterFullscreen === 'function') {
            window.syncGalleryAddressAfterFullscreen()
        }
    } else {
        fullscreenEnabled = true
        $('html body').css("overflow", "hidden").addClass("fullscreen")
        var el = $('html')[0]

        if (typeof document.exitFullscreen == 'function') {
            el.requestFullscreen()

            el.addEventListener("fullscreenchange", function () {
                if (!document.fullscreenElement) {
                    $('html body').css("overflow", "auto").removeClass("fullscreen")
                    fullscreenEnabled = false
                    if (typeof window.syncGalleryAddressAfterFullscreen === 'function') {
                        window.syncGalleryAddressAfterFullscreen()
                    }
                }
            });
        }

    }
}

function exitFullscreen() {
    fullscreenEnabled = false;
    $('html body').css("overflow", "auto").removeClass("fullscreen")
    if (document.fullscreenElement) {
        document.exitFullscreen()
    }
}

function enableDragNDropImagesReordering(albumName) {
    var client = new Backend('');

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
            var beforeTs = $(e.currentTarget).data('ts')
            console.log("dragdrop", "dragged hash", draggedHash, "before", beforeTs, e);

            client.controlSetAlbumImageTime({
                body: {
                    album_name: albumName,
                    image_hash: draggedHash,
                    timestamp: beforeTs
                }
            }, function () {
                $(e.currentTarget).before($("#img" + draggedHash).attr('data-ts', beforeTs))
            }, function (x) {
                alert("Failed to reorder image: " + x.error)
            })

        });

    });

    $("div.chrono-text").each(function () {
        var div = $(this);

        div.on('drop', function (e) {
            e.preventDefault();
            var draggedHash = e.originalEvent.dataTransfer.getData("text/plain");
            var beforeTs = $(e.currentTarget).data('ts')

            console.log("dragdrop-text", "dragged hash", draggedHash, "before", beforeTs, e);

            client.controlSetAlbumImageTime({
                body: {
                    album_name: albumName,
                    image_hash: draggedHash,
                    timestamp: beforeTs
                }
            }, function () {
                $(e.currentTarget).before($("#img" + draggedHash).attr('data-ts', beforeTs))
            }, function (x) {
                alert("Failed to reorder image: " + x.error)
            })

        });

        div.on('dragover', function (e) {
            e.preventDefault();
        });

    });
}

function collectStats(params) {
    params.sw = screen.width
    params.sh = screen.height
    params.px = window.devicePixelRatio
    params.v = window.visitorData.id

    if (document.referrer && new URL(document.referrer).hostname !== window.location.hostname) {
        params.ref = document.referrer
    }

    // mobile portrait mode.
    if (screen.width !== 0 && screen.width <= 576 && window.matchMedia && window.matchMedia("(orientation: portrait)").matches) {
        params.prt = 1
    }

    $.get("/stats", params)
}

function collectThumbVisibility() {
    var visibleSince = {}
    var visibleFor = {}
    var lastFlush = new Date()

    var options = {threshold: 1.0};
    var observer = new IntersectionObserver(function (entries) {
        var now = new Date() - unfocused;
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


// GPS related.

function getDarkColor() {
    var color = '#';
    for (var i = 0; i < 6; i++) {
        color += Math.floor(Math.random() * 12).toString(16);
    }
    return color;
}


// distance returns 2D distance between two points in meters.
function distance(lat1, lon1, lat2, lon2) {
    var toRad = Math.PI / 180
    var dLat = toRad * (lat1 - lat2)
    var dLon = toRad * (lon1 - lon2)

    var a = Math.pow(Math.sin(dLat / 2), 2) + Math.pow(Math.sin(dLon / 2), 2) * Math.cos(toRad * lat1) * Math.cos(toRad * lat2)
    var c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))

    return 6371000 * c
}
