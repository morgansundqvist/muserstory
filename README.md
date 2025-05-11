# muserstory

The project is a command-line interface (CLI) tool designed to help users organize, manage, and enhance their user stories. It enables users to categorize their stories for better organization, allowing for logical grouping and efficient retrieval based on specific categories. Users can propose individual user stories to be stored, with the option to accept or decline them. The tool also supports specifying a markdown file for this storage, offering flexibility in choosing how and where to manage the content.

An important feature is the ability to filter user stories by category, making it easier for users to locate specific stories. Additionally, the CLI provides notifications for potential duplicates, helping to maintain a clean repository of user stories. Users can also export their organized stories into formats such as JSON or CSV, facilitating integration with analytics and reporting tools, enhancing the analytical capabilities of the project.

For team leaders or managers, the ability to assign status labels to user stories—like 'In Progress' or 'Completed'—allows for effective project monitoring and management. Users can submit all their categorized stories to receive comprehensive summaries, providing them with a high-level overview of their projects' features.

The project has the potential to evolve into a sophisticated story management platform, where advanced features such as real-time collaboration, version control, integrations with project management tools, and support for team workflows could be introduced. This would further enhance its utility, making it an indispensable tool for agile development teams and project stakeholders.

Okay, here's a "How to Use" section for your `README.md` based on the `main.go` file you provided. This guide will help users understand how to interact with your `muserstory` CLI tool.

## How to Use `muserstory`

`muserstory` is a Command Line Interface (CLI) tool designed to help you manage user stories, with added support from Large Language Models (LLMs) for tasks like categorization, summarization, and generation.

### Prerequisites

* Ensure you have an OpenAI API key configured in your environment as `OPENAI_API_KEY`, as the tool utilizes the OpenAI LLM service.
* If using the remote functionalities (`push`, `listremote`, `getremote`), ensure the `API_HOST` environment variable is set to the appropriate API endpoint.

### Installation

*(You'll need to add instructions here based on how users will install your CLI. Common methods include:)*

* **Using `go install`:**
    ```bash
    go install github.com/morgansundqvist/muserstory@latest
    ```
* **Downloading a release binary:**
    * (Link to your GitHub releases page if you provide pre-compiled binaries)
* **Building from source:**
    ```bash
    git clone https://github.com/morgansundqvist/muserstory.git
    cd muserstory
    make build
    # Then either move the muserstory binary to your PATH or run as ./muserstory
    ```

### Global Flag

All commands (except `listremote` and `getremote`) require a path to your Markdown file containing user stories. This is specified using the global `--file` (or `-f`) flag.

* `--file <filepath>` or `-f <filepath>`: Path to the markdown file containing user stories.
    * **Default:** `userstories.md` (If the flag is not provided, an error will occur as it's a required flag for most operations).

**Example of using the global flag:**

```bash
muserstory --file my_project_stories.md list
```

If your stories are in a file named `userstories.md` in the current directory, you can often omit the `--file` flag for commands that use it, and it will default. However, the `PersistentPreRunE` logic in your `main.go` currently makes it a *required explicit flag*, meaning you must always provide it even if it's the default `userstories.md`. Consider adjusting this if you want the default to be implicit.

*(Developer Note: The `PersistentPreRunE` checks if `filePath == ""` which means the flag must be explicitly set. If you want an implicit default, this check would need to allow an empty `filePath` and then the `NewUserStoryService` would use the default value if `filePath` is empty after flag parsing.)*

### Commands

Here's a breakdown of the available commands:

#### 1. `add`

Adds a new user story to the specified Markdown file.

* **Usage:** `muserstory --file <filepath> add [story text]`
* **Arguments:**
    * `[story text]`: The full text of the user story you want to add. Must be enclosed in quotes if it contains spaces.
* **Example:**
    ```bash
    muserstory --file project_alpha.md add "As a user, I want to be able to reset my password so that I can regain access to my account if I forget it."
    ```

#### 2. `categorize`

Categorizes all user stories within the specified Markdown file using the LLM service. The categories are typically appended to each user story (e.g., `[Category: Feature Improvement]`).

* **Usage:** `muserstory --file <filepath> categorize`
* **Arguments:** None.
* **Example:**
    ```bash
    muserstory --file my_epic_stories.md categorize
    ```

#### 3. `generate`

Generates a specified number of new user stories based on the existing stories in the file, utilizing the LLM service.

* **Usage:** `muserstory --file <filepath> generate`
* **Flags:**
    * `--num <number>` or `-n <number>`: The number of new user stories to generate.
        * **Default:** `1`
        * Must be a positive integer.
* **Arguments:** None.
* **Example:**
    ```bash
    muserstory --file current_sprint.md generate -n 5
    ```

#### 4. `getremote`

Fetches a specific project and its user stories from a remote server by its unique ID.

* **Usage:** `muserstory getremote --id <project_uuid>`
* **Flags:**
    * `--id <project_uuid>`: (Required) The UUID of the project to fetch from the remote server.
* **Arguments:** None.
* **Note:** This command does *not* use the global `--file` flag as it operates on remote data.
* **Example:**
    ```bash
    muserstory getremote --id "a1b2c3d4-e5f6-7890-1234-567890abcdef"
    ```

#### 5. `list`

Lists all user stories found in the specified Markdown file.

* **Usage:** `muserstory --file <filepath> list`
* **Arguments:** None.
* **Example:**
    ```bash
    muserstory -f user_requirements.md list
    ```

#### 6. `listremote`

Lists all projects available on the remote server.

* **Usage:** `muserstory listremote`
* **Arguments:** None.
* **Note:** This command does *not* use the global `--file` flag as it operates on remote data.
* **Example:**
    ```bash
    muserstory listremote
    ```

#### 7. `push`

Pushes the content of the specified Markdown file to a remote server as a project.

* **Usage:** `muserstory --file <filepath> push`
* **Arguments:** None.
* **Example:**
    ```bash
    muserstory --file release_candidate_stories.md push
    ```

#### 8. `summarize`

Generates and saves a summary of all user stories in the specified Markdown file. The summary is typically added to the top of the file.

* **Usage:** `muserstory --file <filepath> summarize`
* **Arguments:** None.
* **Example:**
    ```bash
    muserstory -f product_backlog.md summarize
    ```

### General Workflow Example

1.  **Initialize your stories file (e.g., `my_project.md`):**
    You can start with an empty file or add a few initial stories manually.
    ```markdown
    ---
    project_name: My Awesome Project
    backend_language: Go
    frontend_language: Svelte
    ---

    # User Stories

    - As a new user, I want to be able to sign up for an account so that I can access the platform.
    ```

2.  **Add a new user story:**
    ```bash
    muserstory --file my_project.md add "As a logged-in user, I want to view my profile so that I can update my information."
    ```

3.  **List all stories:**
    ```bash
    muserstory --file my_project.md list
    ```

4.  **Categorize stories:**
    ```bash
    muserstory --file my_project.md categorize
    ```
    *(Check `my_project.md` to see the added categories)*

5.  **Generate new story ideas:**
    ```bash
    muserstory --file my_project.md generate -n 3
    ```

6.  **Create a summary:**
    ```bash
    muserstory --file my_project.md summarize
    ```
    *(Check `my_project.md` for the new summary section, likely at the top or as defined by your application logic)*

7.  **Push to remote (if configured):**
    ```bash
    muserstory --file my_project.md push
    ```

8.  **List remote projects (to verify):**
    ```bash
    muserstory listremote
    ```

