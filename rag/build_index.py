from langchain_community.embeddings import VertexAIEmbeddings
from langchain_community.vectorstores import Redis
from langchain.text_splitter import CharacterTextSplitter
from dotenv import load_dotenv
import os
from langchain_google_vertexai import VertexAI
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

embeddingModel = VertexAIEmbeddings(model_name="text-embedding-004")
embeddingModel._LanguageModel = VertexAI(model_name="text-embedding-004")
embeddingModel.model_rebuild()


class CodeIngestor:
    def __init__(self, chunkSize=300, chunkOverlap=20):
        self.embeddingModel = embeddingModel
        self.textSplitter = CharacterTextSplitter(
            separator="\n",
            chunk_size=chunkSize,
            chunk_overlap=chunkOverlap
        )
        self.redisVector = None
        

    def readCodeFile(self, filePath):
        with open(filePath, "r") as f:
            return f.read()


    def chunkCode(self, codeText):
        return self.textSplitter.split_text(codeText)


    def embedAndStore(self, chunks, metadata=None):
        embeddings = self.embeddingModel.embed_documents(chunks)
        self.redisVector.add_texts(chunks, metadatas=metadata or [{}])
        return embeddings
    

    def ingestFolder(self, folderPath, fileExtensions=(".py", ".go", ".cpp")):
        indexName = os.path.basename(os.path.normpath(folderPath)) + "_index"
        self.redisVector = Redis(
            redis_url=redisUrl,
            index_name=indexName,
            embedding=self.embeddingModel
        )

        for root, _, files in os.walk(folderPath):
            for file in files:
                if file.endswith(fileExtensions):
                    fullPath = os.path.join(root, file)
                    codeText = self.readCodeFile(fullPath)
                    chunks = self.chunkCode(codeText)
                    metadatas = [{"file": file, "chunk": i} for i in range(len(chunks))]
                    self.embedAndStore(chunks, metadata=metadatas)
                    print(f"Ingested {file} ({len(chunks)} chunks)")
                    

if __name__ == "__main__":
    ingestor = CodeIngestor()
    ingestor.ingestFolder(".")