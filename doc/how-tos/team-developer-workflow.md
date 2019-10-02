# Team Developer Workflow

This section describes an example of how you can
incorporate Pachyderm into your existing enterprise
infrastructure. If you are just starting to use Pachyderm
and not setting up automation for your Pachyderm build
processes, see [Individual Developer Workflow](../how-tos/individual-developer-workflow.html).

Pachyderm is a powerful system for providing data
provenance and scalable processing to data
scientists and engineers. You can make it even
more powerful by integrating it with your existing
continuous integration and continuous deployment (CI/CD)
workflows and systems. This section walks you through the
CI/CD processes that you can create to enable Pachyderm
collaboration within your data science and engineering groups.

As you write code, you test it in containers and
notebooks against sample data that you store in Pachyderm repos or
mount it locally.
Your code runs in development pipelines in Pachyderm.
Pachyderm provides capabilities that assist with day-to-day
development practices, including
the `--build` and `--push-images` flags to the
`pachctl update pipeline` command that enable you to
build and push images to a Docker registry.

Although initial CI setup might require extra effort on your side,
in the long run, it brings significant benefits to your team,
including the following:

* Simplified workflow for data scientists. Data scientists do not need to be
aware of the complexity of the underlying containerized infrastructure. They
can follow an established Git process, and the CI platform takes care of the
Docker build and push process behind the scenes.

* Your CI platform can run additional unit tests against the submitted
code before creating the build.

* Flexibility in tagging Docker images, such as specifying a custom name
and tag or using the commit SHA for tagging.


The following diagram demonstrates automated Pachyderm
development workflow:

![Developer Workflow](../images/d_developer_workflow102.svg)

The automated developer workflow includes the following steps:

1. A new commit triggers a Git hook.

   Typically, Pachyderm users store the following artifacts in a
   Git repository:

   - A Dockerfile that you use to build local images.
   - A `pipeline.json` specification file that you can use in a `Makefile` to
   create local builds, as well as in the CI/CD workflows.
   - The code that performs data transformations.

   A [commit hook in Git](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks)
   for your repository triggers the CI/CD process. It uses the
   information in your pipeline specification for subsequent steps.

2. Build an image.

   Your CI process automatically starts the build of a Docker container
   image based on your code and the Dockerfile.

3. Push the image tagged with commit ID to an image registry.

   Your CI process pushes a Docker image created in Step 2 to your preferred
   image registry. When a data scientist submits their code to Git, a CI
   process uses the Dockerfile in the repository to build, tag with a Git
   commit SHA, and push the container to your image registry.

4. Update the pipeline spec with the tagged image.

   In this step, your CI/CD infrastructure uses your updated `pipeline.json`
   specification and fills in the Git commit
   SHA for the version of the image that must be used in this pipeline.
   Then, it runs the `pachctl update pipeline` command to push the
   updated pipeline specification to Pachyderm. After that,
   Pachyderm pulls a new image from the registry automatically.
   When the production pipeline is updated with the `pipeline.json`
   file that has the correct image tag in it, Pachyderm restarts all pods
   for this pipeline with the new image automatically.