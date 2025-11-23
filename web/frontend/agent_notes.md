
I've added semantic classnames throughout the AudioDetailView component. This should make it easier to select and style elements.
 - The main views (loading, not found, main detail view) have top-level classnames.
 - The audio player and its controls have been given descriptive classnames like `audio-player-section` and `play-pause-button`.
 - The transcript section, including its toolbar and different view modes, now has classnames like `transcript-section`, `transcript-toolbar`, `compact-view-container`, and `timeline-view-container`.
 - Various buttons and badges also have new classnames for easier targeting.
 - I also fixed an issue where some classnames were not being applied correctly.

