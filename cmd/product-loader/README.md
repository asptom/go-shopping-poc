# Product Loader Command

The product loader command orchestrates the complete product ingestion workflow from CSV files into the shopping platform.

## Usage

```bash
./product-loader -csv <path-to-csv> [options]
```

## Command Line Flags

- `-csv string`: Path to CSV file containing products (required)
- `-batch-id string`: Optional batch ID for this ingestion (auto-generated if not provided)
- `-use-cache`: Use cached images when available (default: true)
- `-reset-cache`: Reset image cache before processing (default: false)

## Examples

```bash
# Basic usage with auto-generated batch ID
./product-loader -csv products.csv

# Custom batch ID with cache control
./product-loader -csv products.csv -batch-id "batch-2024-01" -use-cache=true

# Reset cache before processing
./product-loader -csv products.csv -reset-cache=true
```

## Environment Variables

The command requires the following environment variables to be set:

- `DATABASE_URL`: PostgreSQL connection string
- `PRODUCT_CACHE_DIR`: Directory for caching downloaded images
- `MINIO_ENDPOINT`: MinIO S3 endpoint for image storage
- `MINIO_ACCESS_KEY`: MinIO access key
- `MINIO_SECRET_KEY`: MinIO secret key
- `KAFKA_BROKERS`: Kafka broker addresses for event publishing
- `KAFKA_TOPIC`: Kafka topic for product events

## CSV Format

The CSV file must contain the following columns:

```csv
id,name,description,initial_price,final_price,currency,in_stock,color,size,main_image,country_code,image_count,model_number,root_category,category,brand,all_available_sizes,image_urls
```

Example:
```csv
1,"Test Product","Test description",100.00,90.00,USD,true,Red,M,image.jpg,US,1,TEST123,Clothing,Shirts,TestBrand,[],"[""http://example.com/image.jpg""]"
```

## Workflow

1. **Validation**: Validates CSV file format and required fields
2. **Parsing**: Parses CSV records into product structures
3. **Image Processing**: Downloads and caches product images
4. **Storage**: Uploads images to MinIO object storage
5. **Database**: Inserts products and image metadata
6. **Events**: Publishes product lifecycle events to Kafka

## Output

The command provides detailed logging of the ingestion process including:

- Total products processed
- Image download/upload statistics
- Processing duration
- Any errors encountered
- Batch ID for tracking

## Error Handling

- Invalid CSV files are rejected with detailed error messages
- Network failures during image downloads are retried
- Database transaction failures roll back all changes
- Partial failures don't stop the entire batch processing

## Docker

Build and run using Docker:

```bash
# Build
docker build -t product-loader ./cmd/product-loader

# Run (environment variables must be provided)
docker run \
  -e DATABASE_URL=postgres://localhost:5432/shop \
  -e PRODUCT_CACHE_DIR=/cache \
  -e MINIO_ENDPOINT=http://localhost:9000 \
  -e MINIO_ACCESS_KEY=your-access-key \
  -e MINIO_SECRET_KEY=your-secret-key \
  product-loader -csv /path/to/products.csv
```