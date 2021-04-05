# photo_cache_service
Webserver for images and photos with cache of thumbnails and images.

Simple and very fast webserver, which helps you to browse your photoalbums. For comfort work it creates thumbnails and small copies of all images.

INSTALLATION
------------

Can be started from windows or ubuntu. Make sure you download from ./build correct version of the bin file and config.json.
Necessary folder with images and for cache can be set in config.json


REQUIREMENTS
------------

The minimum requirement by photo_cache_service is that you have Web server. Thats all.
If you want to make some changes, you'd better to know that web part are located in ./i folder and implemented into binary with go-bindata (https://github.com/jteeuwen/go-bindata) tool.

QUICK START
-----------

Photo_cache_service comes with a command line tool called "photo_cache_service" that can run webserver on port :9090 and start creating thumbnails.
