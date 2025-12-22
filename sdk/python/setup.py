from setuptools import setup, find_packages

setup(
    name="satgate",
    version="0.3.1",
    description="Python SDK for SatGate - Automatic L402 payments for AI Agents. The Stripe Moment.",
    long_description=open("README.md").read() if __import__("os").path.exists("README.md") else "",
    long_description_content_type="text/markdown",
    author="SatGate Team",
    author_email="contact@satgate.io",
    url="https://github.com/SatGate-io/satgate",
    project_urls={
        "Homepage": "https://satgate.io",
        "Documentation": "https://satgate.io/playground",
        "Repository": "https://github.com/SatGate-io/satgate",
    },
    packages=find_packages(),
    install_requires=[
        "requests>=2.25.0",
    ],
    extras_require={
        "langchain": [
            "langchain>=0.1.0",
            "langchain-core>=0.1.0",
            "pydantic>=2.0.0",
        ],
        "openai": [
            "langchain>=0.1.0",
            "langchain-openai>=0.0.5",
            "pydantic>=2.0.0",
        ],
        "dev": ["pytest", "responses", "flask"],
    },
    python_requires=">=3.8",
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Internet :: WWW/HTTP",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "Topic :: Scientific/Engineering :: Artificial Intelligence",
    ],
    keywords="l402 lightning bitcoin micropayments api ai agents langchain openai gpt",
)

