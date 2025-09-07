#!/usr/bin/env python3
"""
POC RAG Knowledge Base System Architecture Diagram Generator
Generates a comprehensive architecture overview diagram in WEBP format
"""

import matplotlib.pyplot as plt
import matplotlib.patches as patches
from matplotlib.patches import FancyBboxPatch, ConnectionPatch
import numpy as np
from PIL import Image
import io

def create_architecture_diagram():
    # Create figure and axis
    fig, ax = plt.subplots(1, 1, figsize=(16, 12))
    ax.set_xlim(0, 16)
    ax.set_ylim(0, 12)
    ax.axis('off')
    
    # Define colors
    colors = {
        'frontend': '#61DAFB',      # React blue
        'api': '#FF6B6B',           # API Gateway red
        'lambda': '#FF9F43',        # Lambda orange
        'bedrock': '#9B59B6',       # Bedrock purple
        'opensearch': '#3742FA',    # OpenSearch blue
        'storage': '#2ED573',       # S3 green
        'database': '#FFA502',      # DynamoDB orange
        'user': '#A4B0BE',          # User gray
        'flow': '#2F3542'           # Flow arrows dark
    }
    
    # Title
    ax.text(8, 11.5, 'POC RAG Knowledge Base System Architecture', 
            fontsize=20, fontweight='bold', ha='center')
    
    # User layer
    user_box = FancyBboxPatch((1, 9.5), 2, 1, 
                              boxstyle="round,pad=0.1", 
                              facecolor=colors['user'], 
                              edgecolor='black', linewidth=2)
    ax.add_patch(user_box)
    ax.text(2, 10, 'User\n(Browser)', fontsize=10, ha='center', va='center', fontweight='bold')
    
    # Frontend layer
    frontend_box = FancyBboxPatch((6, 9.5), 3, 1, 
                                  boxstyle="round,pad=0.1", 
                                  facecolor=colors['frontend'], 
                                  edgecolor='black', linewidth=2)
    ax.add_patch(frontend_box)
    ax.text(7.5, 10, 'Frontend\n(React App)', fontsize=10, ha='center', va='center', fontweight='bold')
    
    # API Gateway layer
    api_box = FancyBboxPatch((6, 7.5), 3, 1, 
                             boxstyle="round,pad=0.1", 
                             facecolor=colors['api'], 
                             edgecolor='black', linewidth=2)
    ax.add_patch(api_box)
    # AWS API Gateway stencil
    ax.text(6.2, 8.3, 'API', fontsize=8, ha='left', va='center', fontweight='bold', 
            bbox=dict(boxstyle="round,pad=0.2", facecolor='white', alpha=0.8))
    ax.text(7.5, 8, 'API Gateway\n(REST API)', fontsize=10, ha='center', va='center', fontweight='bold')
    
    # Lambda layer
    lambda_box = FancyBboxPatch((6, 5.5), 3, 1, 
                                boxstyle="round,pad=0.1", 
                                facecolor=colors['lambda'], 
                                edgecolor='black', linewidth=2)
    ax.add_patch(lambda_box)
    # AWS Lambda stencil
    ax.text(6.2, 6.3, 'λ', fontsize=12, ha='left', va='center', fontweight='bold', color='white')
    ax.text(7.5, 6, 'Lambda Functions\n(Go Backend)', fontsize=10, ha='center', va='center', fontweight='bold')
    
    # Bedrock Knowledge Base
    bedrock_box = FancyBboxPatch((2, 3.5), 3, 1, 
                                 boxstyle="round,pad=0.1", 
                                 facecolor=colors['bedrock'], 
                                 edgecolor='black', linewidth=2)
    ax.add_patch(bedrock_box)
    # AWS Bedrock stencil
    ax.text(2.2, 4.3, 'BR', fontsize=8, ha='left', va='center', fontweight='bold', 
            bbox=dict(boxstyle="round,pad=0.2", facecolor='white', alpha=0.8))
    ax.text(3.5, 4, 'AWS Bedrock\nKnowledge Base\n(Claude 3 Haiku)', fontsize=9, ha='center', va='center', fontweight='bold', color='white')
    
    # OpenSearch Serverless
    opensearch_box = FancyBboxPatch((10, 3.5), 3, 1, 
                                    boxstyle="round,pad=0.1", 
                                    facecolor=colors['opensearch'], 
                                    edgecolor='black', linewidth=2)
    ax.add_patch(opensearch_box)
    # AWS OpenSearch stencil
    ax.text(10.2, 4.3, 'OS', fontsize=8, ha='left', va='center', fontweight='bold', color='white',
            bbox=dict(boxstyle="round,pad=0.2", facecolor='white', alpha=0.3))
    ax.text(11.5, 4, 'OpenSearch\nServerless\n(Vector Search)', fontsize=9, ha='center', va='center', fontweight='bold', color='white')
    
    # S3 Storage
    s3_box = FancyBboxPatch((2, 1.5), 3, 1, 
                            boxstyle="round,pad=0.1", 
                            facecolor=colors['storage'], 
                            edgecolor='black', linewidth=2)
    ax.add_patch(s3_box)
    # AWS S3 stencil
    ax.text(2.2, 2.3, 'S3', fontsize=8, ha='left', va='center', fontweight='bold', 
            bbox=dict(boxstyle="round,pad=0.2", facecolor='white', alpha=0.8))
    ax.text(3.5, 2, 'Amazon S3\n(Document Storage)', fontsize=10, ha='center', va='center', fontweight='bold', color='white')
    
    # DynamoDB
    dynamo_box = FancyBboxPatch((10, 5.5), 3, 1, 
                                boxstyle="round,pad=0.1", 
                                facecolor=colors['database'], 
                                edgecolor='black', linewidth=2)
    ax.add_patch(dynamo_box)
    # AWS DynamoDB stencil
    ax.text(10.2, 6.3, 'DB', fontsize=8, ha='left', va='center', fontweight='bold', 
            bbox=dict(boxstyle="round,pad=0.2", facecolor='white', alpha=0.8))
    ax.text(11.5, 6, 'DynamoDB\n(Metadata)', fontsize=10, ha='center', va='center', fontweight='bold')
    
    # Data flow arrows
    arrows = [
        # User to Frontend
        ((3, 10), (6, 10)),
        # Frontend to API Gateway
        ((7.5, 9.5), (7.5, 8.5)),
        # API Gateway to Lambda
        ((7.5, 7.5), (7.5, 6.5)),
        # Lambda to Bedrock
        ((6.5, 6), (4.5, 4.5)),
        # Lambda to DynamoDB
        ((8.5, 6), (10.5, 6)),
        # Bedrock to OpenSearch
        ((5, 4), (10, 4)),
        # Bedrock to S3
        ((3.5, 3.5), (3.5, 2.5)),
        # S3 to OpenSearch (ingestion)
        ((5, 2), (10, 3.5)),
    ]
    
    for start, end in arrows:
        arrow = ConnectionPatch(start, end, "data", "data",
                               arrowstyle="->", shrinkA=5, shrinkB=5,
                               mutation_scale=20, fc=colors['flow'], ec=colors['flow'],
                               linewidth=2)
        ax.add_patch(arrow)
    
    # Add flow labels
    ax.text(4.5, 10.3, 'HTTP Requests', fontsize=8, ha='center', style='italic')
    ax.text(8.5, 9, 'REST API', fontsize=8, ha='center', style='italic')
    ax.text(8.5, 7, 'Function Calls', fontsize=8, ha='center', style='italic')
    ax.text(5, 5, 'RAG Query', fontsize=8, ha='center', style='italic', rotation=45)
    ax.text(9.5, 6.3, 'Metadata', fontsize=8, ha='center', style='italic')
    ax.text(7.5, 4.3, 'Vector Search', fontsize=8, ha='center', style='italic')
    ax.text(1.5, 3, 'Document\nRetrieval', fontsize=8, ha='center', style='italic')
    ax.text(7.5, 2.8, 'Document\nIngestion', fontsize=8, ha='center', style='italic', rotation=30)
    
    # Add process flow indicators
    ax.text(0.5, 8, '1. Upload', fontsize=9, fontweight='bold', bbox=dict(boxstyle="round,pad=0.3", facecolor='lightblue'))
    ax.text(0.5, 7.5, '2. Query', fontsize=9, fontweight='bold', bbox=dict(boxstyle="round,pad=0.3", facecolor='lightgreen'))
    ax.text(0.5, 7, '3. Retrieve', fontsize=9, fontweight='bold', bbox=dict(boxstyle="round,pad=0.3", facecolor='lightyellow'))
    ax.text(0.5, 6.5, '4. Generate', fontsize=9, fontweight='bold', bbox=dict(boxstyle="round,pad=0.3", facecolor='lightcoral'))
    
    # Add technology stack info
    tech_info = [
        "Technology Stack:",
        "• Frontend: React + TypeScript",
        "• Backend: Go + AWS Lambda",
        "• API: AWS API Gateway",
        "• AI: AWS Bedrock (Claude 3 Haiku)",
        "• Search: OpenSearch Serverless",
        "• Storage: Amazon S3",
        "• Database: DynamoDB",
        "• Infrastructure: AWS SAM"
    ]
    
    for i, line in enumerate(tech_info):
        weight = 'bold' if i == 0 else 'normal'
        ax.text(14, 8.5 - i*0.3, line, fontsize=8, fontweight=weight,
                bbox=dict(boxstyle="round,pad=0.1", facecolor='white', alpha=0.8) if i == 0 else None)
    
    # Add key features
    features = [
        "Key Features:",
        "• PDF/Text Document Upload",
        "• Vector-based Semantic Search", 
        "• AI-Powered Question Answering",
        "• Real-time Chat Interface",
        "• Serverless Architecture",
        "• Japanese Language Support"
    ]
    
    for i, line in enumerate(features):
        weight = 'bold' if i == 0 else 'normal'
        ax.text(14, 5.5 - i*0.3, line, fontsize=8, fontweight=weight,
                bbox=dict(boxstyle="round,pad=0.1", facecolor='white', alpha=0.8) if i == 0 else None)
    
    plt.tight_layout()
    
    # Save as WEBP
    buffer = io.BytesIO()
    plt.savefig(buffer, format='png', dpi=300, bbox_inches='tight', facecolor='white')
    buffer.seek(0)
    
    # Convert to WEBP
    image = Image.open(buffer)
    image.save('docs/architecture-overview.webp', 'WEBP', quality=90, optimize=True)
    
    plt.close()
    print("✅ Architecture diagram saved as docs/architecture-overview.webp")

if __name__ == "__main__":
    create_architecture_diagram()