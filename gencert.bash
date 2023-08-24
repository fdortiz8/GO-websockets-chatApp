# need to host server as https. this requires that we have certificates 
# can create a self signed certificate using openssl. 
# openSSL in github for installation


# this file will be used to create certificates using openssl
# this file is a bash script to do certificates 


# BEGIN

#!/bin/bash


# try and fix this. the time for this is 1:56-2:02
echo "creating server.key"
openssl genrsa -out server.key 2048 #generate rsa and output as server key
openssl ecparam -genkey -name secp384r1 -out server.key # use secp384rl algorithm
echo "creating server.crt"
openssl req -new -x509 -sha256 -key server.key -out server.crt -batch -days 3650

# these 3 lines when ran create two files server.crt and server.key 
# we can use these two files to encrypt traffic 
# remember to add those files if you are using certificates in your code.
# fetch the certificates from somewhere safe. DO NOT PUSH THEM TO GITHUB
# there are bots out there that will scrape your repository and steal your keys
# git ignore them and DO NOT STORE THEM in your gir repository 