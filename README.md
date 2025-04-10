# Mattermost Feed Plugin

A simple Mattermost plugin to post RSS and Atom feeds into channels.

## Features

-   Subscribe to RSS or Atom feeds in any channel
-   Automatically post new content from feeds to the channel
-   Simple slash commands to manage feed subscriptions
-   Background job fetches feed updates every 20 minutes

## Installation

1. Download the latest release from the [releases page](https://github.com/manybugsdev/mattermost-plugin-feed/releases)
2. Upload the plugin to your Mattermost instance through **System Console > Plugins > Management**
3. Enable the plugin

## Usage

The plugin uses slash commands to manage feeds:

### Add a feed to a channel

```
/feed add https://example.com/feed.xml
```

### List feeds in the current channel

```
/feed list
```

### Remove a feed from a channel

```
/feed del https://example.com/feed.xml
```

## How It Works

-   The plugin creates a bot account that posts updates from feeds
-   A background job runs every 20 minutes to check for new feed items
-   New items are posted to the channel where the feed was added
-   Only items newer than the subscription date are posted

## Development

### Requirements

-   Go 1.22+
-   Make

### Setup

1. Clone the repository

```
git clone https://github.com/manybugsdev/mattermost-plugin-feed.git
cd mattermost-plugin-feed
```

2. Build the plugin

```
make
```

3. Deploy to a local Mattermost development server

```
make deploy
```

### Environment Variables

For deployment:

-   `MM_SERVICESETTINGS_SITEURL` - URL to your Mattermost instance
-   `MM_ADMIN_TOKEN` or `MM_ADMIN_USERNAME`/`MM_ADMIN_PASSWORD` - Authentication credentials

## Building

To build the plugin:

```
make dist
```

This will create a `.tar.gz` file in the `dist/` directory that can be uploaded to Mattermost.

## License

This plugin is licensed under the Apache License 2.0. See [LICENSE](LICENSE) for more information.
