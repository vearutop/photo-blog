#!/bin/bash

curl https://raw.githubusercontent.com/dimsemenov/PhotoSwipe/master/dist/umd/photoswipe.umd.min.js -o photoswipe.umd.min.js
curl https://raw.githubusercontent.com/dimsemenov/PhotoSwipe/master/dist/umd/photoswipe-lightbox.umd.min.js -o photoswipe-lightbox.umd.min.js
curl https://raw.githubusercontent.com/dimsemenov/PhotoSwipe/master/dist/photoswipe.css -o photoswipe.css
curl https://raw.githubusercontent.com/dimsemenov/photoswipe-dynamic-caption-plugin/main/dist/photoswipe-dynamic-caption-plugin.umd.min.js -o photoswipe-dynamic-caption-plugin.umd.min.js
curl https://raw.githubusercontent.com/dimsemenov/photoswipe-dynamic-caption-plugin/main/photoswipe-dynamic-caption-plugin.css -o photoswipe-dynamic-caption-plugin.css
gzip -9 *.css
gzip -9 *.js
