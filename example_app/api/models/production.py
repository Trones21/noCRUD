from django.db import models
from rest_framework import serializers, viewsets
from .universe import Universe


class Production(models.Model):
    TYPE_CHOICES = [
        ("film", "Film"),
        ("show", "TV Show"),
        ("web", "Web Series"),
        ("short", "Short"),
        ("promo", "Promo Clip"),
        ("doc", "Documentary"),
    ]

    title = models.CharField(max_length=255)
    year = models.IntegerField()
    type = models.CharField(max_length=10, choices=TYPE_CHOICES)

    # Media
    image = models.URLField(blank=True)
    trailer_url = models.URLField(blank=True)
    description = models.TextField(blank=True)

    # Source metadata
    is_official = models.BooleanField(null=True)
    distributor = models.CharField(
        max_length=255, blank=True
    )  # e.g., AMC, Netflix, YouTube
    external_id = models.CharField(
        max_length=100, blank=True
    )  # e.g., IMDB ID, YouTube Playlist ID

    universe = models.ForeignKey(
        Universe,
        null=True,
        blank=True,
        on_delete=models.SET_NULL,
        related_name="productions",
    )

    def __str__(self):
        return f"{self.title} ({self.year})"

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "production"


class ProductionSerializer(serializers.ModelSerializer):
    class Meta:
        model = Production
        fields = "__all__"


class ProductionViewSet(viewsets.ModelViewSet):
    queryset = Production.objects.all()
    serializer_class = ProductionSerializer
