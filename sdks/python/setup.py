from setuptools import setup, find_packages

setup(
    name="chronos-guard-sdk",
    version="1.0.0",
    packages=find_packages(),
    install_requires=[
        "grpcio>=1.60.0",
        "protobuf>=4.25.0",
    ],
)