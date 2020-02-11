/*
Package state_gen implements the service to manage both hot and cold states in DB. It allows
both the hot and cold state to be stored in different intervals and sections. It uses beacon
blocks playback so one can play back blocks and compute desired state at any input slot.
*/
package stategen
