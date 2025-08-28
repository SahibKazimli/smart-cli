from langchain_redis import RedisVectorStore
from langchain.text_splitter import CharacterTextSplitter
from dotenv import load_dotenv
import os
from langchain_google_vertexai import VertexAIEmbeddings
from vertexai import init

"""Prototyped version of the RAG capability. I'll be using redis
and generating index names dynamically based on the folder the user
is in. """


load_dotenv()
redisUrl = os.getenv("REDIS_URL", "redis://localhost:6379")
projectId = os.getenv("GCP_PROJECT_ID", "my-default-project")

init(
    project=projectId,
    location="us-central1"
)

embeddingModel = VertexAIEmbeddings(model_name="text-embedding-005")


class CodeIngestor:
    def __init__(self, chunkSize=400, chunkOverlap=200):
        self.embeddingModel = embeddingModel
        self.textSplitter = CharacterTextSplitter(
            separator="\n",
            chunk_size=chunkSize,
            chunk_overlap=chunkOverlap
        )
        self.redisVector = None
        self.ignore_dirs = {'venv', '.venv', '.git', '__pycache__', 'node_modules'}
        self.include_extensions = {'.py', '.go', '.cpp', '.js', '.ts', '.java', '.sh'}


    def readCodeFile(self, filePath):
        with open(filePath, "r") as f:
            return f.read()


    def chunkCode(self, codeText):
        return self.textSplitter.split_text(codeText)


    def embedAndStore(self, chunks, metadata=None):
        # For langchain-redis, add_texts is the correct method.
        # It internally handles embedding if not already done.
        self.redisVector.add_texts(chunks, metadatas=metadata or [{}])


    def ingestFolder(self, folderPath, fileExtensions=None):
        # Convert the list of extensions to a tuple for file.endswith()
        if fileExtensions is None:
            extensions_tuple = tuple(self.include_extensions)
        else:
            # Ensure the input is a tuple, even if a list was passed
            extensions_tuple = tuple(fileExtensions)

        if not extensions_tuple: # Handle case where no extensions are specified
            print("No file extensions specified for ingestion.")
            return

        indexName = os.path.basename(os.path.normpath(folderPath)) + "_index"
        print(f"Indexing into Redis with index name: {indexName}")

        
        self.redisVector = RedisVectorStore(
            redis_url=redisUrl,
            index_name=indexName,
            embeddings=self.embeddingModel 
        )

        for root, dirs, files in os.walk(folderPath, topdown=True):
            # Modify dirs in-place to prevent os.walk from descending into ignored directories
            dirs[:] = [d for d in dirs if d not in self.ignore_dirs]

            for file in files:
                # Check if the file extension is in our allowed tuple
                # file.endswith() accepts a tuple of strings
                if file.lower().endswith(extensions_tuple):
                    fullPath = os.path.join(root, file)
                    print(f"Ingesting: {fullPath}")
                    try:
                        codeText = self.readCodeFile(fullPath)
                        chunks = self.chunkCode(codeText)
                        # Use relative path for metadata
                        metadatas = [{"file": os.path.relpath(fullPath, folderPath), "chunk": i} for i in range(len(chunks))]
                        self.embedAndStore(chunks, metadata=metadatas)
                        print(f"Ingested {fullPath} ({len(chunks)} chunks)")
                    except Exception as e:
                        print(f"Error ingesting {fullPath}: {e}")
                        

if __name__ == "__main__":
    load_dotenv()
    cwd = os.getcwd()  # wherever the user triggers this script from
    print(f"\n--- Ingesting from working directory: {cwd} ---")
    
    ingestor = CodeIngestor()
    ingestor.ingestFolder(cwd, fileExtensions=['.py', '.go'])