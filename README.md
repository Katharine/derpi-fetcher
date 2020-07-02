derpi-fetcher
=============

Download images from Derpibooru.

Originally built to fetch training data for
[Aerial](https://twitter.com/AerialDraws). The code quality is low because
it was a quick hack at the time.

You can fetch a binary release for your machine from the
[Releases](https://github.com/Katharine/derpi-fetcher/releases) page.

## Usage

Basic usage is just `./fetcher <some derpi query>`, e.g. `./fetcher "artist:myself"`.

Images will be dropped in **the current working directory**, in folders named after the
artist(s) of the images.

In full:
```
./fetcher [flags] <query>
  --filter-id int
        filter ID to use (defaults to 'Everything', 100073 is 'Default') (default 56027)
  --workers int
        number of concurrent downloads (default 100)
```
