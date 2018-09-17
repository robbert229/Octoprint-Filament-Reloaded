# OctoPrint-FilamentReloaded-Server
This plugin is a fork of the the original [Octoprint-FilamentReloaded](https://github.com/kontakt/Octoprint-Filament-Reloaded)
 which is in turn a fork of [Octoprint-Filament](https://github.com/MoonshineSG/Octoprint-Filament). I forked the plugin because I wanted to take a different approach to sensing filament.

If you are running octoprint on a raspberry pi this plugin is not for you. If you are instead running octoprint on an old linux desktop, or have a more advanced setup this plugin may be of value to you.  

## Required Hardware

### Linux Computer

This plugin

### Arduino

The arduino is used to perform the GPIO, and in tern exposes that to a linux machine over the serial connection.

![Arduino](./docs/arduino.png)

## Features

## Installation

## Configuration


## Why

I use a *retired* nas running [coreos](https://coreos.com) to run [octoprint](https://octoprint.org/). It runs multiple instances inside of [docker](https://www.docker.com/) containers using [docker-octoprint](https://github.com/QuantumObject/docker-octoprint). 

Linux desktops, and media servers do not have any GPIO and as such I needed a way of utilizing GPIO over USB. This solution is very beta.

## License

All of the python code written is licensed under GPL-3.0, while the protobuf code, and go code is licensed under MIT. I intend to replace the original "borrowed" code eventually, and when I do I will license the new python code under MIT as well.
