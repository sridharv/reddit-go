// Package reddit provides wrappers to the Reddit API for Go.
//
// Please read the reddit API documentation first before reading these docs.
// Some useful links are:
//
//  * https://github.com/reddit/reddit/wiki/API
//  * https://github.com/reddit/reddit/wiki/JSON
//  * https://github.com/reddit/reddit/wiki/OAuth2-Quick-Start-Example
//  * https://github.com/reddit/reddit/wiki/OAuth2
//  * https://www.reddit.com/dev/api
//
// This currently only supports OAuth for script apps. It does not support
// refreshing OAuth tokens. It provides the following:
//
//  * Code to save and load authorization credentials (client id, client secret, etc).
//  * A simple API to obtain and store an OAuth token for a script app using these credentials.
//  * An API to perform GET requests using the obtained token.
//  * An API to stream listings.
//
// Please see the package examples for details on how to use the above functionality.
package reddit
