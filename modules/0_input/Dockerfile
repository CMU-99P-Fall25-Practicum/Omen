FROM python:3.12.11-bookworm

WORKDIR /app
COPY inputvalidator.py .
RUN chmod +x inputvalidator.py

# install dependencies
RUN pip install pydantic typing

ENTRYPOINT ["python", "inputvalidator.py"]
CMD []

# execute with something like: 
# docker run --rm -v ./good.json:/input/good.json omen-input-validator:latest /input/good.json