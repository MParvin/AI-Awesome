## Project Categorization

| Dimension | Classification |
|-----------|----------------|
| **Type** | Curated Awesome List / Resource Index |
| **Domain** | Artificial Intelligence, Machine Learning, Deep Learning |
| **Primary Use Case** | Discovery & reference for AI/ML/DL tools, frameworks, models, and applications |
| **Target Audience** | AI/ML engineers, researchers, developers, hobbyists, enterprises adopting AI |
| **Scope** | Broad — covers entire AI/ML ecosystem (foundational frameworks → applications) |
| **Maintenance** | Auto-generated via [awesome-stars](https://github.com/mparvin/awesome-stars); last updated 2026-08-07 |
| **Format** | Markdown table of contents with GitHub repo links, star counts, one-line descriptions |

---

### Key Topical Clusters (derived from listed repos)

| Cluster | Representative Repos |
|---------|----------------------|
| **Foundational Frameworks** | `tensorflow/tensorflow`, `huggingface/transformers`, `pytorch/pytorch` (implied), `deepspeedai/DeepSpeed` |
| **LLM Inference & Serving** | `vllm-project/vllm`, `ggml-org/llama.cpp`, `ollama/ollama`, `nomic-ai/gpt4all`, `bentoml/OpenLLM` |
| **Agent Frameworks & Platforms** | `langchain-ai/langchain`, `langchain-ai/langgraph`, `microsoft/autogen`, `crewAIInc/crewAI`, `openai/openai-agents-python`, `ag2ai/ag2` |
| **Coding Agents / AI Dev Toolsops | `anthropics/claude-code`, `openai/codex`, `cline/cline`, `Aider-AI/aider`, `continuedev/continue`, `TabbyML/tabby`, `Pythagora-io/gpt-pilot` |
| **RAG / Vector Search / Knowledge** | `milvus-io/milvus`, `qdrant/qdrant`, `pgvector/pgvector`, `langchain-ai/langgraph`, `Cinnamon/kotaemon`, `HKUDS/LightRAG`, `eosphoros-ai/DB-GPT` |
| **Local-First / Self-Hosted AI | `janhq/jan`, `mudler/LocalAI`, `zylon-ai/private-gpt`, `Mintplex-Labs/anything-llm`, `open-webui/open-webui` |
| **Multimodal / Generative Media** | `AUTOMATIC1111/stable-diffusion-webui`, `Comfy-Org/ComfyUI`, `Stability-AI/generative-models`, `guoyww/AnimateDiff`, `ace-step/ACE-Step`, `coqui-ai/TTS`, `myshell-ai/OpenVoice` |
| **Fine-Tuning / PEFT** | `hiyouga/LlamaFactory`, `microsoft/LoRA`, `artidoro/qlora`, `axolotl-ai-cloud/axolotl` |
| **Specialized Agents** | `TauricResearch/TradingAgents` (finance), `GreyDGL/PentestGPT` (security), `assafelovic/gpt-researcher` (research), `browser-use/browser-use` (web automation) |
| **MCP / Tooling Ecosystem** | `modelcontextprotocol/servers`, `microsoft/playwright-mcp`, `Flux159/mcp-server-kubernetes`, `zcaceres/markdownify-mcp` |
| **Infrastructure / Ops** | `gpustack/gpustack`, `volcano-sh/volcano`, `kai-scheduler/KAI-Scheduler`, `docker/genai-stack` |
| **Datasets & Education** | `awesomedata/awesome-public-datasets`, `microsoft/generative-ai-for-beginners`, `rasbt/LLMs-from-scratch` |

---

### Tech Stack Indicators (from repo metadata)

- **Languages**: Python (dominant), C/C++ (inference engines), Rust (performance-critical), TypeScript/JS (UIs), Go, Zig
- **Hardware Targets**: CUDA, Metal (Apple Silicon), ROCm, CPU-only, mobile/edge (Android, LiteRT)
- **Deployment**: Docker, Kubernetes, local desktop, browser, CLI, VS Code extensions, self-hosted servers
- **Protocols**: OpenAI-compatible APIs, MCP (Model Context Protocol), SSE/HTTP, gRPC