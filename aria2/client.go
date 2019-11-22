package aria2

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
)

type Status string

const (
	// StatusActive for currently downloading/seeding downloads.
	StatusActive Status = "active"
	// StatusWaiting for downloads in the queue, download is not started.
	StatusWaiting Status = "waiting"
	// StatusPaused for paused downloads.
	StatusPaused Status = "paused"
	// StatusError for downloads that were stopped because of error.
	StatusError Status = "error"
	// StatusComplete for stopped and completed downloads.
	StatusComplete Status = "complete"
	// StatusRemoved for the downloads removed by user.
	StatusRemoved Status = "removed"
)

type Uri struct {
	// 'used' if the URI is in use. 'waiting' if the URI is still waiting in the queue.
	Status string `json:"status"`
	// URI
	Uri string `json:"uri"`
}

type File struct {
	// Index of the file, starting at 1, in the same order as files appear in the multi-file torrent.
	Index int `json:"index,string"`
	// File path.
	Path string `json:"path"`
	// File size in bytes.
	Length int `json:"length,string"`
	// Completed length of this file in bytes.
	// Please note that it is possible that sum of completedLength is less than the completedLength returned
	// by the aria2.tellStatus() method. This is because completedLength in aria2.getFiles() only includes completed pieces.
	// On the other hand, completedLength in aria2.tellStatus() also includes partially completed pieces.
	CompletedLength int `json:"completedLength,string"`
	// true if this file is selected by --select-file option.
	// If --select-file is not specified or this is single-file torrent or not a torrent download at all,
	// this value is always true. Otherwise false.
	Selected bool `json:"selected,string"`
	// Returns a list of URIs for this file.
	Uris []*Uri `json:"uris"`
}

type Bittorrent struct {
	// List of lists of announce URIs.
	// If the torrent contains announce and no announce-list, announce is converted to the announce-list format.
	AnnounceList [][]string `json:"announceList"`
	// The comment of the torrent. comment.utf-8 is used if available.
	Comment string `json:"comment,omitempty"`
	// The creation time of the torrent. The value is an integer since the epoch, measured in seconds.
	CreationDate int `json:"creationDate"`
	// File mode of the torrent. The value is either single or multi.
	Mode string `json:"mode"`
	// Struct which contains data from Info dictionary. It contains following keys.
	Info struct {
		// name in info dictionary. name.utf-8 is used if available.
		Name string `json:"name"`
	} `json:"info"`
}

type TaskStatus struct {
	// GID of the download.
	Gid string `json:"gid"`
	// Status of task.
	Status Status `json:"status"`
	// Total length of the download in bytes.
	TotalLength int `json:"totalLength,string"`
	// Completed length of the download in bytes.
	CompletedLength int `json:"completedLength,string"`
	// Uploaded length of the download in bytes.
	UploadLength int `json:"uploadLength,string"`
	// Hexadecimal representation of the download progress. The highest bit corresponds to the piece at index 0.
	// Any set bits indicate loaded pieces, while unset bits indicate not yet loaded and/or missing pieces.
	// Any overflow bits at the end are set to zero. When the download was not started yet,
	// this key will not be included in the response.
	Bitfield string `json:"bitfield"`
	// Download speed of this download measured in bytes/sec.
	DownloadSpeed int `json:"downloadSpeed,string"`
	// Upload speed of this download measured in bytes/sec.
	UploadSpeed int `json:"uploadSpeed,string"`
	// InfoHash. BitTorrent only.
	InfoHash string `json:"infoHash"`
	// The number of seeders aria2 has connected to. BitTorrent only.
	NumSeeders int `json:"numSeeders,string"`
	// true if the local endpoint is a seeder. Otherwise false. BitTorrent only.
	Seeder bool `json:"seeder,string"`
	// Piece length in bytes.
	PieceLength int `json:"pieceLength,string"`
	// The number of pieces.
	NumPieces int `json:"numPieces,string"`
	// The number of peers/servers aria2 has connected to.
	Connections int `json:"connections,string"`
	// The code of the last error for this item, if any. This value is only available for stopped/completed downloads.
	ErrorCode int `json:"errorCode,string,omitempty"`
	// The (hopefully) human readable error message associated to errorCode.
	ErrorMessage string `json:"errorMessage,omitempty"`
	// List of GIDs which are generated as the result of this download.
	// For example, when aria2 downloads a Metalink file, it generates downloads described
	// in the Metalink (see the --follow-metalink option). This value is useful to track auto-generated downloads.
	// If there are no such downloads, this key will not be included in the response.
	FollowedBy []string `json:"followedBy,omitempty"`
	// The reverse link for followedBy. A download included in followedBy has this object's GID in its following value.
	Following string `json:"following"`
	// GID of a parent download. Some downloads are a part of another download.
	// For example, if a file in a Metalink has BitTorrent resources,
	// the downloads of ".torrent" files are parts of that parent.
	// If this download has no parent, this key will not be included in the response.
	BelongsTo string `json:"belongsTo,omitempty"`
	// Directory to save files.
	Dir string `json:"dir"`
	// Returns the list of files.
	Files []*File `json:"files"`
	// Struct which contains information retrieved from the .torrent (file).
	// BitTorrent only. It contains following keys.
	Bittorrent *Bittorrent `json:"bittorrent,omitempty"`
	// The number of verified number of bytes while the files are being hash checked.
	// This key exists only when this download is being hash checked.
	VerifiedLength int `json:"verifiedLength,string,omitempty"`
	// true if this download is waiting for the hash check in a queue.
	// This key exists only when this download is in the queue.
	VerifyIntegrityPending bool `json:"verifyIntegrityPending,string,omitempty"`
}

type Peer struct {
	// Percent-encoded peer ID.
	PeerId string `json:"peerId"`
	// IP address of the peer.
	IP string `json:"ip"`
	// Port number of the peer.
	Port int `json:"port,string"`
	// Hexadecimal representation of the download progress of the peer.
	// The highest bit corresponds to the piece at index 0.
	// Set bits indicate the piece is available and unset bits indicate the piece is missing.
	// Any spare bits at the end are set to zero.
	Bitfield string `json:"bitfield"`
	// true if aria2 is choking the peer. Otherwise false.
	AmChoking bool `json:"amChoking,string"`
	// true if the peer is choking aria2. Otherwise false.
	PeerChoking bool `json:"peerChoking,string"`
	// Download speed (byte/sec) that this client obtains from the peer.
	DownloadSpeed int `json:"downloadSpeed,string"`
	// Upload speed(byte/sec) that this client uploads to the peer.
	UploadSpeed int `json:"uploadSpeed,string"`
	// true if this peer is a seeder. Otherwise false.
	Seeder bool `json:"seeder,string"`
}

type Client struct {
	hc       *http.Client
	secret   string
	endpoint string
}

func New(endpoint, secret string) *Client {
	return &Client{
		hc:       &http.Client{},
		secret:   secret,
		endpoint: endpoint,
	}
}

func (c *Client) do(method string, args []interface{}, reply interface{}) error {
	opts := make([]interface{}, 0, len(args)+2)
	if c.secret != "" {
		opts = append(opts, fmt.Sprintf("token:%s", c.secret))
	}
	opts = append(opts, args...)
	b, err := encodeClientRequest(method, opts)
	if err != nil {
		return err
	}
	resp, err := c.hc.Post(c.endpoint, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeClientResponse(resp.Body, reply)
}

// This method adds a new download. uris is an array of HTTP/FTP/SFTP/BitTorrent URIs (strings) pointing to the same resource.
// If you mix URIs pointing to different resources, then the download may fail or be corrupted without aria2 complaining.
// When adding BitTorrent Magnet URIs, uris must have only one element and it should be BitTorrent Magnet URI.
// options is a struct and its members are pairs of option name and value.
func (c *Client) AddUri(uris []string, options ...option) (string, error) {
	method := "aria2.addUri"
	args := []interface{}{uris, newOptions().applyOption(options...)}
	var gid string
	if err := c.do(method, args, &gid); err != nil {
		return "", err
	}
	return gid, nil
}

// This method adds a BitTorrent download by uploading a ".torrent" file.
// If you want to add a BitTorrent Magnet URI, use the AddUri() method instead.
// uris is an array of URIs (string). uris is used for Web-seeding.
// For single file torrents, the URI can be a complete URI pointing to the resource;
// if URI ends with /, name in torrent file is added. For multi-file torrents,
// name and path in torrent are added to form a URI for each file.
func (c *Client) AddTorrent(torrent []byte, uris []string, options ...option) (string, error) {
	method := "aria2.addTorrent"
	if uris == nil {
		uris = []string{}
	}
	args := []interface{}{
		base64.StdEncoding.EncodeToString(torrent),
		uris,
		newOptions().applyOption(options...),
	}
	var gid string
	if err := c.do(method, args, &gid); err != nil {
		return "", err
	}
	return gid, nil
}

// This method adds a Metalink download by uploading a ".metalink" file.
func (c *Client) AddMetalink(metalink []byte, options ...option) (string, error) {
	method := "aria2.addMetalink"
	args := []interface{}{
		base64.StdEncoding.EncodeToString(metalink),
		newOptions().applyOption(options...),
	}
	var gid string
	if err := c.do(method, args, &gid); err != nil {
		return "", err
	}
	return gid, nil
}

// This method removes the download denoted by gid (string).
// If the specified download is in progress, it is first stopped.
// The status of the removed download becomes removed.
// This method returns GID of removed download.
func (c *Client) Remove(gid string) error {
	method := "aria2.remove"
	var s string
	return c.do(method, []interface{}{gid}, &s)
}

// This method removes the download denoted by gid.
// This method behaves just like Remove() except that this method removes
// the download without performing any actions which take time,
// such as contacting BitTorrent trackers to unregister the download first.
func (c *Client) ForceRemove(gid string) error {
	method := "aria2.forceRemove"
	var s string
	return c.do(method, []interface{}{gid}, &s)
}

// This method pauses the download denoted by gid (string).
// The status of paused download becomes paused.
// If the download was active, the download is placed in the front of waiting queue.
// While the status is paused, the download is not started.
// To change status to waiting, use the Unpause() method.
func (c *Client) Pause(gid string) error {
	method := "aria2.pause"
	var s string
	return c.do(method, []interface{}{gid}, &s)
}

// This method is equal to calling aria2.pause() for every active/waiting download.
func (c *Client) PauseAll() error {
	method := "aria2.pauseAll"
	var s string
	return c.do(method, nil, &s)
}

// This method pauses the download denoted by gid.
// This method behaves just like Pause() except that this method pauses
// downloads without performing any actions which take time,
// such as contacting BitTorrent trackers to unregister the download first.
func (c *Client) ForcePause(gid string) error {
	method := "aria2.forcePause"
	var s string
	return c.do(method, []interface{}{gid}, &s)
}

// This method is equal to calling aria2.forcePause() for every active/waiting download.
func (c *Client) ForcePauseAll() error {
	method := "aria2.forcePauseAll"
	var s string
	return c.do(method, nil, &s)
}

// This method changes the status of the download denoted by gid (string)
// from paused to waiting, making the download eligible to be restarted.
func (c *Client) Unpause(gid string) error {
	method := "aria2.unpause"
	var s string
	return c.do(method, []interface{}{gid}, &s)
}

// This method is equal to calling aria2.unpause() for every paused download.
func (c *Client) UnpauseAll() error {
	method := "aria2.unpauseAll"
	var s string
	return c.do(method, nil, &s)
}

// This method returns the progress of the download denoted by gid (string).
func (c *Client) TellStatus(gid string) (*TaskStatus, error) {
	method := "aria2.tellStatus"
	taskStatus := new(TaskStatus)
	if err := c.do(method, []interface{}{gid}, taskStatus); err != nil {
		return nil, err
	}
	return taskStatus, nil
}

// This method returns the URIs used in the download denoted by gid.
func (c *Client) GetUris(gid string) ([]*Uri, error) {
	method := "aria2.getUris"
	var uris []*Uri
	if err := c.do(method, []interface{}{gid}, &uris); err != nil {
		return nil, err
	}
	return uris, nil
}

// This method returns the file list of the download denoted by gid.
func (c *Client) GetFiles(gid string) ([]*File, error) {
	method := "aria2.getFiles"
	var files []*File
	if err := c.do(method, []interface{}{gid}, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// This method returns a list peers of the download denoted by gid (string). This method is for BitTorrent only.
func (c *Client) GetPeers(gid string) ([]*Peer, error) {
	method := "aria2.getPeers"
	var peers []*Peer
	if err := c.do(method, []interface{}{gid}, &peers); err != nil {
		return nil, err
	}
	return peers, nil
}
