package hls

/*
	https://tools.ietf.org/html/draft-pantos-http-live-streaming-23#page-7

	NOTES

	Although implied by implementations, the HLS spec does not require some tags to appear in
	a specific position in the playlist. For example, the EXT-X-ENDLIST tag can appear anywhere
	in the playlist, and this is explicit in the RFC.

	VOD playlist are immutable,
	EVENT playlists are an append-only.
	LIVE playlists are a sliding window.

	The server can only increment sequence tags and EXT-X-ENDLIST, as well as push and pop segments.

	Any timing information and play order in the media playlists are coincidental. The RFC says
	that the order of the segments dictates which order they are played in.

	Each segment has a sequence and discontinuity sequence number. Both properties are
	computed.
*/
