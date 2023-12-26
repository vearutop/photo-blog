:::{lang=en}
## Adding First Album

Once you've started the `photo-blog` application, it will create a storage directory `photo-blog-data` in your 
current directory. This new directory will contain all state of the app.

The app will serve on port `8008`, you open http://127.0.0.1:8008/ if you run it locally. 

![empty.png](empty.png)

It might be a good idea to set admin password, to restrict access to albums management. Once the password is set,
browser will asc you for name and password, name can be anything, password should be the one you've set.

Application settings are available under gear button. You can set your site title and some other parameters there.

Then click on a film button to add an album.

![add-album.png](add-album.png)

Name can not be changed later (though you can always create a copy of album with a new name and delete the old one) 
and will act as a part of URL to the new album.

Name is started with a random prefix by default, this is to hide private albums from people that can guess a name. 
Remove this random prefix if you intend to make the album public. Please note, URL-protected private albums are not 
sufficiently secure for really sensitive data. Private URL can leak to search engines from your browser, it can be 
overshared by people you gave access without your control.


![edit-album.png](edit-album.jpg)

Once you've created an album, you will be redirected to a page where cat edit more details and upload pictures.

You can also upload [GPX](https://en.wikipedia.org/wiki/GPS_Exchange_Format) files, they will be shown on the map too.

Click "Back to album" to open album page.

![view-album.png](view-album.jpg)

Photos, that have geotags will be shown on the map. 

Get back to editing album/adding more photos with a pencil button on top.

Click home button to open main page.

## Main Page

![main-page.jpg](main-page.jpg)

Each album (that does not have "Hidden" option enabled) will be listed on the main page. 
Private albums are only listed if you're logged in as admin.

Login button or admin controls are shown at the bottom of the page.

:::
