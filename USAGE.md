### Directory Structure 

| Directory      | Description / Status                              |
|----------------|------------------------------------------|
| `python/`      | Python Implementation - **Serial version complete, currently working on parallelization**    |
| `go/`          | Go Implementation - **Super Basic - still playing around with the patterns, not as feature complete as the python implementation**     |
| `example-app/` | Example Application for running tests against   |
| `example-runner/` | This is what your runner will look like after you start building it out a bit. |

### Using this Tool 

Unlike many libraries and frameworks out there, you are meant to copy the source code directly over to start. If you compare the python implementation with the example-runner, you will see that the example-runner has all the same files, it just also has the actual test files that point against the example app.

I recommend looking through the example-runner before you start building out your own runner to get an idea of some of the patterns possible.

**See the readme inside the runner for the flags and examples.**

### Notes: 
There aren't many authentication options yet, each implementation only has the auth options for the application I was creating the implementation for, but that's the beauty of having the source code! Just open up the `APIClient` and modify it to meet your needs. (same go for everything else in the utils folder... this is really the core of the application)


