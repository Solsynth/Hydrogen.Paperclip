# Hydrogen.Paperclip

Paperclip is the unified attachment service for all hydrogen services.
It contains file metadata compute, instant upload, calculating hashing, multi destination, media info and more features!

## Features

Paperclip store and processing uploaded files with pipeline flow.
When a user try to upload files. The file will store in local first for media processing.

Then the server will publish a message into the message queue.
And the background consumer will start dealing with the uploaded files.

The background consumer will hash the file and merge the files with same hashcode.
The background consumer will decode the image and generate ratio and read more info from image file too.

After the processing done. The consumer will upload the file to the permanent storage like a s3 bucket and remove local cache.
While the processing, the file record in database will marked to the temporary and load file from the temporary storage.
When the processing done, the file record will be updated.

### Supported Destinations

- Local filesystem
- S3 compilable bucket
