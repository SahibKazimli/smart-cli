from langchain_community.embeddings import VertexAIEmbeddings
from langchain_community.vectorstores import Redis
from langchain.text_splitter import CharacterTextSplitter
from dotenv import load_dotenv
import os

"""Prototyped version of the RAG capability. I'll be using redis
and generating index names dynamically based on the folder the user
is in. """


load_dotenv()
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
embedding_model = VertexAIEmbeddings(model_name="text-embedding-004")

class CodeIngestor:
    def __init__(self, embedding_model_name=embedding_model, chunk_size=300, chunk_overlap=20):
        self.embedding_model = VertexAIEmbeddings(model_name=embedding_model_name)
        self.text_splitter = CharacterTextSplitter(
            separator="\n",
            chunk_size=chunk_size,
            chunk_overlap=chunk_overlap
        )
        self.redis_vector = None


    def read_code_file(self, file_path):
        with open(file_path, "r") as f:
            return f.read()


    def chunk_code(self, code_text):
        return self.text_splitter.split_text(code_text)


    def embed_and_store(self, chunks, metadata=None):
        # Embed each chunk
        embeddings = self.embedding_model.embed_documents(chunks)
        # Store in Redis with metadata
        self.redis_vector.add_texts(chunks, metadatas=metadata or [{}])
        return embeddings
    

    def ingest_folder(self, folder_path, file_extensions=(".py", ".go", ".cpp")):
        index_name = os.path.basename(os.path.normpath(folder_path)) + "_index"
        self.redis_vector = Redis(
            redis_url=REDIS_URL,
            index_name=index_name,
            embedding=embedding_model
        )
        
        for root, _, files in os.walk(folder_path):
            for file in files:
                if file.endswith(file_extensions):
                    full_path = os.path.join(root, file)
                    code_text = self.read_code_file(full_path)
                    chunks = self.chunk_code(code_text)
                    # Add file info in metadata
                    metadatas = [{"file": file, "chunk": i} for i in range(len(chunks))]
                    self.embed_and_store(chunks, metadata=metadatas)
                    print(f"Ingested {file} ({len(chunks)} chunks)")