from django.db import models
from rest_framework import serializers, viewsets
from .universe import Universe

class Production(models.Model):
    TYPE_CHOICES = [('film', 'Film'), ('show', 'TV Show')]
    title = models.CharField(max_length=255)
    year = models.IntegerField()
    type = models.CharField(max_length=10, choices=TYPE_CHOICES)
    image = models.URLField(blank=True)
    description = models.TextField(blank=True)
    universe = models.ForeignKey(Universe, null=True, blank=True, on_delete=models.SET_NULL, related_name='productions')

    def __str__(self):
        return f"{self.title} ({self.year})"

class ProductionSerializer(serializers.ModelSerializer):
    class Meta:
        model = Production
        fields = '__all__'

class ProductionViewSet(viewsets.ModelViewSet):
    queryset = Production.objects.all()
    serializer_class = ProductionSerializer