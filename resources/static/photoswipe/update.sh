#!/bin/bash

rm *.gz

curl https://raw.githubusercontent.com/dimsemenov/PhotoSwipe/master/dist/umd/photoswipe.umd.min.js -o photoswipe.umd.min.js
curl https://raw.githubusercontent.com/dimsemenov/PhotoSwipe/master/dist/umd/photoswipe-lightbox.umd.min.js -o photoswipe-lightbox.umd.min.js
curl https://raw.githubusercontent.com/dimsemenov/PhotoSwipe/master/dist/photoswipe.css -o photoswipe.css
curl https://raw.githubusercontent.com/dimsemenov/photoswipe-dynamic-caption-plugin/main/dist/photoswipe-dynamic-caption-plugin.umd.min.js -o photoswipe-dynamic-caption-plugin.umd.min.js
curl https://raw.githubusercontent.com/dimsemenov/photoswipe-dynamic-caption-plugin/main/photoswipe-dynamic-caption-plugin.css -o photoswipe-dynamic-caption-plugin.css

curl https://cdn.jsdelivr.net/gh/dimsemenov/photoswipe-video-plugin@5e32d6589df53df2887900bcd55267d72aee57a6/dist/photoswipe-video-plugin.esm.min.js -o photoswipe-video-plugin.esm.min.js
curl https://cdn.jsdelivr.net/gh/arnowelzel/photoswipe-auto-hide-ui@1.0.1/photoswipe-auto-hide-ui.esm.min.js -o photoswipe-auto-hide-ui.esm.min.js
curl https://cdn.jsdelivr.net/gh/dpet23/photoswipe-slideshow@v2.0.0/photoswipe-slideshow.esm.min.js -o photoswipe-slideshow.esm.min.js

npm install esm2umd

npx esm2umd PhotoSwipeSlideshow photoswipe-slideshow.esm.min.js > photoswipe-slideshow.umd.min.js
rm photoswipe-slideshow.esm.min.js

npx esm2umd PhotoSwipeAutoHideUI photoswipe-auto-hide-ui.esm.min.js > photoswipe-auto-hide-ui.umd.min.js
rm photoswipe-auto-hide-ui.esm.min.js

npx esm2umd PhotoSwipeVideoPlugin photoswipe-video-plugin.esm.min.js > photoswipe-video-plugin.umd.min.js
rm photoswipe-video-plugin.esm.min.js

rm -rf ./node_modules
rm package.json
rm package-lock.json

gzip -9 *.css
gzip -9 *.js
